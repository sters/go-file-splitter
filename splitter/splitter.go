package splitter

import (
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var ErrTypeCast = errors.New("failed to cast to GenDecl")

type TestFunction struct {
	Name               string
	FuncDecl           *ast.FuncDecl
	Comments           *ast.CommentGroup
	StandaloneComments []*ast.CommentGroup // Comments that appear before the function but are not doc comments
	Imports            []*ast.ImportSpec
	Package            string
}

func SplitTestFiles(directory string) error {
	testFiles, err := findTestFiles(directory)
	if err != nil {
		return fmt.Errorf("failed to find test files: %w", err)
	}

	for _, file := range testFiles {
		if err := processTestFile(file); err != nil {
			return fmt.Errorf("failed to process %s: %w", file, err)
		}
	}

	return nil
}

func findTestFiles(directory string) ([]string, error) {
	var testFiles []string

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, "_test.go") {
			testFiles = append(testFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return testFiles, nil
}

func processTestFile(filename string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	tests, hasRemainingContent := extractTestFunctions(node)
	if len(tests) == 0 {
		return nil
	}

	outputDir := filepath.Dir(filename)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, test := range tests {
		snakeCaseName := testNameToSnakeCase(test.Name)
		outputFileName := snakeCaseName + "_test.go"

		// Check if the generated filename would conflict with the original
		if outputFileName == filepath.Base(filename) {
			outputFileName = "splitted_" + outputFileName
		}

		outputFile := filepath.Join(outputDir, outputFileName)
		if err := writeTestFile(outputFile, test, fset); err != nil {
			return fmt.Errorf("failed to write test file %s: %w", outputFile, err)
		}
		fmt.Printf("Created: %s\n", outputFile)
	}

	// Only delete the original file if there's no remaining content
	if !hasRemainingContent {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("failed to delete original file %s: %w", filename, err)
		}
		fmt.Printf("Deleted original: %s\n", filename)
	} else {
		// Remove extracted tests from the original file
		// removeExtractedTests will delete the file if it becomes empty
		if err := removeExtractedTests(filename, tests, fset); err != nil {
			return fmt.Errorf("failed to update original file %s: %w", filename, err)
		}
		// Check if file still exists after removal
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			fmt.Printf("Preserved original: %s (contains non-split tests or helper functions)\n", filename)
		}
	}

	return nil
}

func removeExtractedTests(filename string, extractedTests []TestFunction, fset *token.FileSet) error {
	// Re-parse the file to get a clean AST
	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Create a map of extracted test names for quick lookup
	extractedNames := make(map[string]bool)
	for _, test := range extractedTests {
		extractedNames[test.Name] = true
	}

	// Filter out the extracted tests from declarations
	var newDecls []ast.Decl
	hasRemainingContent := false
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if extractedNames[fn.Name.Name] {
				// Skip this function as it was extracted
				continue
			}
			// This is a function that wasn't extracted
			hasRemainingContent = true
			newDecls = append(newDecls, decl)
		} else if genDecl, ok := decl.(*ast.GenDecl); ok {
			// Check if this is an import declaration
			if genDecl.Tok == token.IMPORT {
				// Keep imports only if there's other remaining content
				// We'll add them back later if needed
				continue
			}
			// Non-import GenDecl (types, vars, consts)
			hasRemainingContent = true
			newDecls = append(newDecls, decl)
		}
	}

	// Also track positions of extracted functions to remove orphaned comments
	extractedPositions := make(map[token.Pos]bool)
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if extractedNames[fn.Name.Name] {
				extractedPositions[fn.Pos()] = true
			}
		}
	}

	// Remove orphaned comments by filtering node.Comments
	node.Comments = filterOrphanedComments(node, extractedNames)

	// If there's no remaining content, delete the file
	if !hasRemainingContent || len(newDecls) == 0 {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("failed to delete empty file: %w", err)
		}
		fmt.Printf("Deleted original (now empty): %s\n", filename)

		return nil
	}

	// Filter imports to only include used ones
	usedImports := findUsedImportsInDecls(newDecls, node.Imports)

	// Re-add only used imports if there's remaining content
	var finalDecls []ast.Decl
	if len(usedImports) > 0 {
		importDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(usedImports)),
		}
		for i, imp := range usedImports {
			importDecl.Specs[i] = imp
		}
		finalDecls = append(finalDecls, importDecl)
	}
	finalDecls = append(finalDecls, newDecls...)
	node.Decls = finalDecls

	// Format and write back to file
	var buf strings.Builder
	if err := format.Node(&buf, fset, node); err != nil {
		return fmt.Errorf("failed to format code: %w", err)
	}

	if err := os.WriteFile(filename, []byte(buf.String()), 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func filterOrphanedComments(node *ast.File, extractedNames map[string]bool) []*ast.CommentGroup {
	var filteredComments []*ast.CommentGroup
	for _, cg := range node.Comments {
		if shouldKeepComment(cg, node, extractedNames) {
			filteredComments = append(filteredComments, cg)
		}
	}

	return filteredComments
}

func shouldKeepComment(cg *ast.CommentGroup, node *ast.File, extractedNames map[string]bool) bool {
	// Check if this comment group was associated with an extracted function
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !extractedNames[fn.Name.Name] {
			continue
		}

		// If the comment is the doc comment for an extracted function, remove it
		if fn.Doc == cg {
			return false
		}

		// Check if comment belongs to the extracted function (including before and after comments)
		if isTestSpecificComment(cg, fn, node.Decls) {
			return false
		}

		// Also remove comments inside the extracted function body
		if fn.Body != nil && cg.Pos() >= fn.Body.Lbrace && cg.End() <= fn.Body.Rbrace {
			return false
		}
	}

	return true
}

// This only returns true for standalone comments that should be included with the test.
func isTestSpecificComment(cg *ast.CommentGroup, fn *ast.FuncDecl, allDecls []ast.Decl) bool {
	// Skip if comment is inside the function body itself - these are handled automatically
	if fn.Body != nil && cg.Pos() >= fn.Body.Lbrace && cg.End() <= fn.Body.Rbrace {
		// Comments inside the function body are handled automatically by format.Node
		return false
	}

	// Skip if this comment is inside another function body
	for _, otherDecl := range allDecls {
		if otherFn, ok := otherDecl.(*ast.FuncDecl); ok && otherFn != fn {
			if otherFn.Body != nil && cg.Pos() >= otherFn.Body.Lbrace && cg.End() <= otherFn.Body.Rbrace {
				return false
			}
		}
	}

	// Find the function's position in the declarations
	fnIndex := -1
	for i, decl := range allDecls {
		if decl == fn {
			fnIndex = i

			break
		}
	}

	if fnIndex == -1 {
		return false
	}

	// Check comments before the function
	if cg.End() >= fn.Pos() {
		// Comments after functions stay in the original file
		if fn.Body != nil && cg.Pos() > fn.Body.Rbrace {
			return false
		}

		return false
	}

	// Find the previous declaration
	var prevDecl ast.Decl
	prevDeclEnd := token.Pos(0)
	for i := fnIndex - 1; i >= 0; i-- {
		if decl := allDecls[i]; decl != nil {
			prevDecl = decl
			if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Body != nil {
				prevDeclEnd = funcDecl.Body.Rbrace
			} else {
				prevDeclEnd = decl.End()
			}

			break
		}
	}

	// If there's a previous declaration, check which function the comment is closer to
	if prevDecl != nil {
		distToPrevDecl := cg.Pos() - prevDeclEnd
		distToCurrentFunc := fn.Pos() - cg.End()

		// If comment is closer to previous declaration, it belongs to that
		if distToPrevDecl < distToCurrentFunc {
			return false
		}
	}

	// Comment belongs to this function if it's after the previous declaration
	// and reasonably close to the function (within 50 lines)
	return cg.Pos() > prevDeclEnd && fn.Pos()-cg.End() < token.Pos(50*80)
}

func extractTestFunctions(node *ast.File) ([]TestFunction, bool) {
	tests := make([]TestFunction, 0, len(node.Decls))
	hasRemainingContent := false

	// Map function positions to indices for finding standalone comments
	funcPositions := make(map[token.Pos]int)
	for i, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			funcPositions[fn.Pos()] = i
		}
	}

	for _, decl := range node.Decls {
		fn, isFuncDecl := decl.(*ast.FuncDecl)
		if !isFuncDecl {
			if _, ok := decl.(*ast.GenDecl); ok {
				// Type declarations, constants, variables should be preserved
				hasRemainingContent = true
			}

			continue
		}

		if !strings.HasPrefix(fn.Name.Name, "Test") || fn.Recv != nil {
			if fn.Recv == nil {
				// Non-test functions (helper functions) should be preserved
				hasRemainingContent = true
			}

			continue
		}

		// Check if the character after "Test" (and any underscores) is uppercase
		nameAfterTest := strings.TrimPrefix(fn.Name.Name, "Test")
		nameAfterTest = strings.TrimLeft(nameAfterTest, "_")

		// Skip if empty or starts with lowercase (e.g., Test_foo)
		if len(nameAfterTest) == 0 || unicode.IsLower(rune(nameAfterTest[0])) {
			hasRemainingContent = true

			continue
		}

		// Find standalone comments that belong to this function
		var standaloneComments []*ast.CommentGroup
		for _, cg := range node.Comments {
			// Skip if this is the doc comment
			if cg == fn.Doc {
				continue
			}

			// Only include comments that are specifically for this test function
			// and not for other functions or general file comments
			if isTestSpecificComment(cg, fn, node.Decls) {
				standaloneComments = append(standaloneComments, cg)
			}
		}

		test := TestFunction{
			Name:               fn.Name.Name,
			FuncDecl:           fn,
			Comments:           fn.Doc,
			StandaloneComments: standaloneComments,
			Imports:            node.Imports,
			Package:            node.Name.Name,
		}
		tests = append(tests, test)
	}

	return tests, hasRemainingContent
}

func writeTestFile(filename string, test TestFunction, fset *token.FileSet) error {
	// Build declarations: imports first, then the test function
	var decls []ast.Decl

	// Find which imports are actually used in this test function
	usedImports := findUsedImports(test.FuncDecl, test.Imports)

	// Add import declarations if there are any used imports
	if len(usedImports) > 0 {
		decls = append(decls, &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(usedImports)),
		})
		// Copy import specs
		for i, imp := range usedImports {
			genDecl, ok := decls[0].(*ast.GenDecl)
			if !ok {
				return ErrTypeCast
			}
			genDecl.Specs[i] = imp
		}
	}

	// Add the test function with its comments
	if test.Comments != nil {
		test.FuncDecl.Doc = test.Comments
	}
	decls = append(decls, test.FuncDecl)

	// Create an AST file with the test function and imports
	astFile := &ast.File{
		Name:     &ast.Ident{Name: test.Package},
		Decls:    decls,
		Comments: test.StandaloneComments, // Include standalone comments
	}

	// Format the source code
	var buf strings.Builder
	if err := format.Node(&buf, fset, astFile); err != nil {
		return fmt.Errorf("failed to format code: %w", err)
	}

	// Write the formatted code to file
	if err := os.WriteFile(filename, []byte(buf.String()), 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func findUsedImports(fn *ast.FuncDecl, allImports []*ast.ImportSpec) []*ast.ImportSpec {
	usedPackages := make(map[string]bool)

	// Always include "testing" package for test functions
	usedPackages["testing"] = true

	// Walk through the function body to find used packages
	ast.Inspect(fn, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			// e.g., fmt.Println, strings.HasPrefix
			if ident, ok := x.X.(*ast.Ident); ok {
				usedPackages[ident.Name] = true
			}
		case *ast.CallExpr:
			// Check for type assertions and conversions that might use imported types
			if ident, ok := x.Fun.(*ast.Ident); ok {
				usedPackages[ident.Name] = true
			}
		case *ast.Ident:
			// Check for types from imported packages
			// This is a simplified check - might need refinement for complex cases
			if x.Obj == nil && x.Name != "" {
				// Could be a package-level identifier
				usedPackages[x.Name] = true
			}
		}

		return true
	})

	// Filter imports to only include used ones
	var result []*ast.ImportSpec
	for _, imp := range allImports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Get the package name (last part of import path or alias)
		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		// Check if this import should be included
		switch {
		case importPath == "testing" && usedPackages["testing"]:
			result = append(result, imp)
		case usedPackages[pkgName]:
			result = append(result, imp)
		case strings.Contains(importPath, "testify/assert") && usedPackages["assert"]:
			result = append(result, imp)
		case strings.Contains(importPath, "testify/require") && usedPackages["require"]:
			result = append(result, imp)
		case strings.Contains(importPath, "testify/suite") && usedPackages["suite"]:
			result = append(result, imp)
		}
	}

	return result
}

func findUsedImportsInDecls(decls []ast.Decl, allImports []*ast.ImportSpec) []*ast.ImportSpec {
	usedPackages := make(map[string]bool)

	// Always include "testing" package if there are test-related functions
	for _, decl := range decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fn.Name.Name, "Test") || strings.HasPrefix(fn.Name.Name, "Benchmark") || strings.HasPrefix(fn.Name.Name, "Example") {
				usedPackages["testing"] = true

				break
			}
		}
	}

	// Walk through all declarations to find used packages
	for _, decl := range decls {
		ast.Inspect(decl, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				// e.g., fmt.Println, strings.HasPrefix
				if ident, ok := x.X.(*ast.Ident); ok {
					usedPackages[ident.Name] = true
				}
			case *ast.CallExpr:
				// Check for type assertions and conversions that might use imported types
				if ident, ok := x.Fun.(*ast.Ident); ok {
					usedPackages[ident.Name] = true
				}
			case *ast.Ident:
				// Check for types from imported packages
				// This is a simplified check - might need refinement for complex cases
				if x.Obj == nil && x.Name != "" {
					// Could be a package-level identifier
					usedPackages[x.Name] = true
				}
			}

			return true
		})
	}

	// Filter imports to only include used ones
	var result []*ast.ImportSpec
	for _, imp := range allImports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Get the package name (last part of import path or alias)
		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		// Check if this import should be included
		switch {
		case importPath == "testing" && usedPackages["testing"]:
			result = append(result, imp)
		case usedPackages[pkgName]:
			result = append(result, imp)
		case strings.Contains(importPath, "testify/assert") && usedPackages["assert"]:
			result = append(result, imp)
		case strings.Contains(importPath, "testify/require") && usedPackages["require"]:
			result = append(result, imp)
		case strings.Contains(importPath, "testify/suite") && usedPackages["suite"]:
			result = append(result, imp)
		}
	}

	return result
}

// getCommonAbbreviations returns common abbreviations and Go keywords that should not be split.
func getCommonAbbreviations() []string {
	return []string{
		"ID", "UUID", "URL", "URI", "API", "HTTP", "HTTPS", "JSON", "XML", "CSV",
		"SQL", "DB", "TCP", "UDP", "IP", "DNS", "SSH", "TLS", "SSL", "JWT",
		"AWS", "GCP", "CPU", "GPU", "RAM", "ROM", "IO", "EOF", "TTL", "CDN",
		"HTML", "CSS", "JS", "MD5", "SHA", "RSA", "AES", "UTF", "ASCII",
		"CRUD", "REST", "RPC", "GRPC", "MQTT", "AMQP", "SMTP", "IMAP", "POP",
		"SDK", "CLI", "GUI", "UI", "UX", "OS", "VM", "PDF", "PNG", "JPG", "GIF",
	}
}

// matchesAbbreviation checks if the substring at position i matches any known abbreviation.
func matchesAbbreviation(runes []rune, i int) (string, int) {
	commonAbbreviations := getCommonAbbreviations()
	for _, abbr := range commonAbbreviations {
		if i+len(abbr) > len(runes) {
			continue
		}

		substr := string(runes[i : i+len(abbr)])
		if strings.ToUpper(substr) != abbr {
			continue
		}

		// Check if it's a word boundary (next char is uppercase or end of string)
		atWordBoundary := i+len(abbr) == len(runes) ||
			(i+len(abbr) < len(runes) && unicode.IsUpper(runes[i+len(abbr)]))

		if atWordBoundary {
			return abbr, len(abbr)
		}
	}

	return "", 0
}

// shouldAddUnderscore checks if an underscore should be added before the current character.
func shouldAddUnderscore(runes []rune, i int, result []rune) bool {
	if i == 0 || !unicode.IsUpper(runes[i]) {
		return false
	}

	if len(result) == 0 || result[len(result)-1] == '_' {
		return false
	}

	// Uppercase followed by lowercase (e.g., "Name" in "UserName")
	if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
		return true
	}

	// Lowercase followed by uppercase (e.g., "N" in "userName")
	if i > 0 && unicode.IsLower(runes[i-1]) {
		return true
	}

	return false
}

func testNameToSnakeCase(name string) string {
	if !strings.HasPrefix(name, "Test") {
		return strings.ToLower(name)
	}

	name = strings.TrimPrefix(name, "Test")

	// Remove leading underscores first
	name = strings.TrimLeft(name, "_")

	if name == "" {
		return "test"
	}

	// Check if the entire name (after Test prefix) is a common abbreviation
	commonAbbreviations := getCommonAbbreviations()
	for _, abbr := range commonAbbreviations {
		if strings.ToUpper(name) == abbr {
			return strings.ToLower(name)
		}
	}

	// Process the name character by character, handling abbreviations
	result := make([]rune, 0, len(name)*2)
	runes := []rune(name)

	for i := 0; i < len(runes); i++ {
		// Check if current position starts with a known abbreviation
		if abbr, length := matchesAbbreviation(runes, i); abbr != "" {
			// Add underscore before abbreviation if needed
			if i > 0 && len(result) > 0 && result[len(result)-1] != '_' {
				result = append(result, '_')
			}
			// Add the abbreviation in lowercase
			for _, r := range strings.ToLower(abbr) {
				result = append(result, r)
			}
			i += length - 1 // -1 because the loop will increment

			continue
		}

		// Handle regular character
		r := runes[i]
		if shouldAddUnderscore(runes, i, result) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}

	resultStr := string(result)
	if resultStr == "" {
		return "test"
	}

	return resultStr
}
