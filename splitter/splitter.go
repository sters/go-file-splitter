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

type PublicFunction struct {
	Name               string
	FuncDecl           *ast.FuncDecl
	Comments           *ast.CommentGroup
	StandaloneComments []*ast.CommentGroup
	Imports            []*ast.ImportSpec
	Package            string
}

type PublicDeclaration struct {
	GenDecl  *ast.GenDecl
	Comments *ast.CommentGroup
	Package  string
	Imports  []*ast.ImportSpec
}

func SplitPublicFunctions(directory string) error {
	goFiles, err := findGoFiles(directory)
	if err != nil {
		return fmt.Errorf("failed to find go files: %w", err)
	}

	for _, file := range goFiles {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if err := processGoFile(file); err != nil {
			return fmt.Errorf("failed to process %s: %w", file, err)
		}
	}

	return nil
}

func SplitTestFunctions(directory string) error {
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

func findGoFiles(directory string) ([]string, error) {
	var goFiles []string

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			goFiles = append(goFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return goFiles, nil
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

func processGoFile(filename string) error {
	fset := token.NewFileSet()
	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	publicFuncs := extractPublicFunctions(node)
	publicDecls := extractPublicDeclarations(node)

	if len(publicFuncs) == 0 && len(publicDecls) == 0 {
		return nil
	}

	outputDir := filepath.Dir(filename)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write public functions to individual files
	for _, fn := range publicFuncs {
		snakeCaseName := functionNameToSnakeCase(fn.Name)
		outputFileName := snakeCaseName + ".go"
		outputFile := filepath.Join(outputDir, outputFileName)

		if err := writePublicFunction(outputFile, fn, fset); err != nil {
			return fmt.Errorf("failed to write function file %s: %w", outputFile, err)
		}
		fmt.Printf("Created: %s\n", outputFile)

		// Find and split corresponding test file
		testFile := findCorrespondingTestFile(filename, fn.Name)
		if testFile != "" {
			if err := splitTestForFunction(testFile, fn.Name, outputDir); err != nil {
				fmt.Printf("Warning: failed to split test for %s: %v\n", fn.Name, err)
			}
		}
	}

	// Write public const/var declarations to common.go
	if len(publicDecls) > 0 {
		commonFile := filepath.Join(outputDir, "common.go")
		if err := writeCommonFile(commonFile, publicDecls, node.Name.Name, node.Imports, fset); err != nil {
			return fmt.Errorf("failed to write common.go: %w", err)
		}
		fmt.Printf("Created: %s\n", commonFile)
	}

	// Update original file to keep only private content
	if err := updateOriginalFile(filename, publicFuncs, publicDecls, fset); err != nil {
		return fmt.Errorf("failed to update original file: %w", err)
	}

	return nil
}

func processTestFile(filename string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	tests := extractTestFunctions(node)
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
		if err := writeTestFunction(outputFile, test, fset); err != nil {
			return fmt.Errorf("failed to write test file %s: %w", outputFile, err)
		}
		fmt.Printf("Created: %s\n", outputFile)
	}

	// Remove extracted tests from original file
	if err := removeExtractedTests(filename, tests, fset); err != nil {
		return fmt.Errorf("failed to update original file %s: %w", filename, err)
	}

	return nil
}

func extractPublicFunctions(node *ast.File) []PublicFunction {
	var publicFuncs []PublicFunction

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}

		// Check if function is public (starts with uppercase)
		if !unicode.IsUpper(rune(fn.Name.Name[0])) {
			continue
		}

		var standaloneComments []*ast.CommentGroup
		for _, cg := range node.Comments {
			if cg == fn.Doc {
				continue
			}
			if isFunctionSpecificComment(cg, fn, node.Decls) {
				standaloneComments = append(standaloneComments, cg)
			}
		}

		publicFunc := PublicFunction{
			Name:               fn.Name.Name,
			FuncDecl:           fn,
			Comments:           fn.Doc,
			StandaloneComments: standaloneComments,
			Imports:            node.Imports,
			Package:            node.Name.Name,
		}
		publicFuncs = append(publicFuncs, publicFunc)
	}

	return publicFuncs
}

func extractPublicDeclarations(node *ast.File) []PublicDeclaration {
	var publicDecls []PublicDeclaration

	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok == token.IMPORT {
			continue
		}

		// Check if this declaration contains any public const/var
		hasPublic := false
		for _, spec := range genDecl.Specs {
			switch s := spec.(type) {
			case *ast.ValueSpec:
				for _, name := range s.Names {
					if unicode.IsUpper(rune(name.Name[0])) {
						hasPublic = true

						break
					}
				}
			}
			if hasPublic {
				break
			}
		}

		if hasPublic {
			publicDecl := PublicDeclaration{
				GenDecl:  genDecl,
				Comments: genDecl.Doc,
				Package:  node.Name.Name,
				Imports:  node.Imports,
			}
			publicDecls = append(publicDecls, publicDecl)
		}
	}

	return publicDecls
}

func extractTestFunctions(node *ast.File) []TestFunction {
	var tests []TestFunction

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}

		if !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}

		// Check if the character after "Test" (and any underscores) is uppercase
		nameAfterTest := strings.TrimPrefix(fn.Name.Name, "Test")
		nameAfterTest = strings.TrimLeft(nameAfterTest, "_")

		// Skip if empty or starts with lowercase
		if len(nameAfterTest) == 0 || unicode.IsLower(rune(nameAfterTest[0])) {
			continue
		}

		var standaloneComments []*ast.CommentGroup
		for _, cg := range node.Comments {
			if cg == fn.Doc {
				continue
			}
			if isFunctionSpecificComment(cg, fn, node.Decls) {
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

	return tests
}

type TestFunction struct {
	Name               string
	FuncDecl           *ast.FuncDecl
	Comments           *ast.CommentGroup
	StandaloneComments []*ast.CommentGroup
	Imports            []*ast.ImportSpec
	Package            string
}

func isFunctionSpecificComment(cg *ast.CommentGroup, fn *ast.FuncDecl, allDecls []ast.Decl) bool {
	// Skip if comment is inside the function body
	if fn.Body != nil && cg.Pos() >= fn.Body.Lbrace && cg.End() <= fn.Body.Rbrace {
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
	// and reasonably close to the function
	return cg.Pos() > prevDeclEnd && fn.Pos()-cg.End() < token.Pos(50*80)
}

func writePublicFunction(filename string, fn PublicFunction, fset *token.FileSet) error {
	var decls []ast.Decl

	// Find which imports are actually used
	usedImports := findUsedImports(fn.FuncDecl, fn.Imports)

	// Add import declarations if there are any used imports
	if len(usedImports) > 0 {
		importDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(usedImports)),
		}
		for i, imp := range usedImports {
			importDecl.Specs[i] = imp
		}
		decls = append(decls, importDecl)
	}

	// Add the function with its comments
	if fn.Comments != nil {
		fn.FuncDecl.Doc = fn.Comments
	}
	decls = append(decls, fn.FuncDecl)

	// Create an AST file
	astFile := &ast.File{
		Name:     &ast.Ident{Name: fn.Package},
		Decls:    decls,
		Comments: fn.StandaloneComments,
	}

	// Format and write to file
	if err := formatAndWriteFile(filename, astFile, fset); err != nil {
		return err
	}

	return nil
}

func writeTestFunction(filename string, test TestFunction, fset *token.FileSet) error {
	var decls []ast.Decl

	// Find which imports are actually used
	usedImports := findUsedImports(test.FuncDecl, test.Imports)

	// Add import declarations if there are any used imports
	if len(usedImports) > 0 {
		importDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(usedImports)),
		}
		for i, imp := range usedImports {
			importDecl.Specs[i] = imp
		}
		decls = append(decls, importDecl)
	}

	// Add the test function with its comments
	if test.Comments != nil {
		test.FuncDecl.Doc = test.Comments
	}
	decls = append(decls, test.FuncDecl)

	// Create an AST file
	astFile := &ast.File{
		Name:     &ast.Ident{Name: test.Package},
		Decls:    decls,
		Comments: test.StandaloneComments,
	}

	// Format and write to file
	if err := formatAndWriteFile(filename, astFile, fset); err != nil {
		return err
	}

	return nil
}

func writeCommonFile(filename string, decls []PublicDeclaration, pkgName string, imports []*ast.ImportSpec, fset *token.FileSet) error {
	var astDecls []ast.Decl

	// Collect all used imports from declarations
	usedPackages := make(map[string]bool)
	for _, decl := range decls {
		ast.Inspect(decl.GenDecl, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if ident, ok := x.X.(*ast.Ident); ok {
					usedPackages[ident.Name] = true
				}
			}

			return true
		})
	}

	// Filter and add imports
	var usedImports []*ast.ImportSpec
	for _, imp := range imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		var pkgNameFromImport string
		if imp.Name != nil {
			pkgNameFromImport = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgNameFromImport = parts[len(parts)-1]
		}

		if usedPackages[pkgNameFromImport] {
			usedImports = append(usedImports, imp)
		}
	}

	if len(usedImports) > 0 {
		importDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(usedImports)),
		}
		for i, imp := range usedImports {
			importDecl.Specs[i] = imp
		}
		astDecls = append(astDecls, importDecl)
	}

	// Add all public declarations
	for _, decl := range decls {
		astDecls = append(astDecls, decl.GenDecl)
	}

	// Create an AST file
	astFile := &ast.File{
		Name:  &ast.Ident{Name: pkgName},
		Decls: astDecls,
	}

	// Format and write to file
	if err := formatAndWriteFile(filename, astFile, fset); err != nil {
		return err
	}

	return nil
}

func formatAndWriteFile(filename string, astFile *ast.File, fset *token.FileSet) error {
	var buf strings.Builder
	if err := format.Node(&buf, fset, astFile); err != nil {
		return fmt.Errorf("failed to format code: %w", err)
	}

	if err := os.WriteFile(filename, []byte(buf.String()), 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func updateOriginalFile(filename string, extractedFuncs []PublicFunction, extractedDecls []PublicDeclaration, fset *token.FileSet) error {
	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Create maps for quick lookup
	extractedFuncNames := make(map[string]bool)
	for _, fn := range extractedFuncs {
		extractedFuncNames[fn.Name] = true
	}

	extractedDeclPtrs := make(map[*ast.GenDecl]bool)
	for _, decl := range extractedDecls {
		extractedDeclPtrs[decl.GenDecl] = true
	}

	// Filter declarations
	var newDecls []ast.Decl
	hasRemainingContent := false

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Keep private functions and methods
			if !extractedFuncNames[d.Name.Name] {
				newDecls = append(newDecls, decl)
				hasRemainingContent = true
			}
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue // We'll re-add imports later if needed
			}
			// Keep private declarations
			if !extractedDeclPtrs[d] {
				// Check if this declaration has any private members
				hasPrivate := false
				for _, spec := range d.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range vs.Names {
							if !unicode.IsUpper(rune(name.Name[0])) {
								hasPrivate = true

								break
							}
						}
					}
				}
				if hasPrivate {
					newDecls = append(newDecls, decl)
					hasRemainingContent = true
				}
			}
		}
	}

	// If no remaining content, delete the file
	if !hasRemainingContent || len(newDecls) == 0 {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("failed to delete empty file: %w", err)
		}
		fmt.Printf("Deleted original (now empty): %s\n", filename)

		return nil
	}

	// Find used imports in remaining declarations
	usedImports := findUsedImportsInDecls(newDecls, node.Imports)

	// Re-add only used imports
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

	// Format and write back
	var buf strings.Builder
	if err := format.Node(&buf, fset, node); err != nil {
		return fmt.Errorf("failed to format code: %w", err)
	}

	if err := os.WriteFile(filename, []byte(buf.String()), 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Updated original: %s (preserved private content)\n", filename)

	return nil
}

func removeExtractedTests(filename string, extractedTests []TestFunction, fset *token.FileSet) error {
	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Create a map of extracted test names
	extractedNames := make(map[string]bool)
	for _, test := range extractedTests {
		extractedNames[test.Name] = true
	}

	// Filter out the extracted tests
	var newDecls []ast.Decl
	hasRemainingContent := false
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if extractedNames[fn.Name.Name] {
				continue
			}
			hasRemainingContent = true
			newDecls = append(newDecls, decl)
		} else if genDecl, ok := decl.(*ast.GenDecl); ok {
			if genDecl.Tok == token.IMPORT {
				continue
			}
			hasRemainingContent = true
			newDecls = append(newDecls, decl)
		}
	}

	// If no remaining content, delete the file
	if !hasRemainingContent || len(newDecls) == 0 {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("failed to delete empty file: %w", err)
		}
		fmt.Printf("Deleted original (now empty): %s\n", filename)

		return nil
	}

	// Find used imports in remaining declarations
	usedImports := findUsedImportsInDecls(newDecls, node.Imports)

	// Re-add only used imports
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

	// Format and write back
	var buf strings.Builder
	if err := format.Node(&buf, fset, node); err != nil {
		return fmt.Errorf("failed to format code: %w", err)
	}

	if err := os.WriteFile(filename, []byte(buf.String()), 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Preserved original: %s (contains non-split tests or helper functions)\n", filename)

	return nil
}

func findCorrespondingTestFile(filename string, functionName string) string {
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	base = strings.TrimSuffix(base, ".go")
	testFile := filepath.Join(dir, base+"_test.go")

	if _, err := os.Stat(testFile); err == nil {
		return testFile
	}

	return ""
}

func splitTestForFunction(testFile string, functionName string, outputDir string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse test file: %w", err)
	}

	// Find test functions that match the public function name
	var matchingTests []TestFunction
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}

		// Check if test name contains the function name
		if strings.Contains(fn.Name.Name, functionName) {
			var standaloneComments []*ast.CommentGroup
			for _, cg := range node.Comments {
				if cg == fn.Doc {
					continue
				}
				if isFunctionSpecificComment(cg, fn, node.Decls) {
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
			matchingTests = append(matchingTests, test)
		}
	}

	// Write matching tests to new file
	if len(matchingTests) > 0 {
		snakeCaseName := functionNameToSnakeCase(functionName)
		outputFileName := snakeCaseName + "_test.go"
		outputFile := filepath.Join(outputDir, outputFileName)

		// Write all matching tests to the same file
		if err := writeTestsToFile(outputFile, matchingTests, fset); err != nil {
			return fmt.Errorf("failed to write test file: %w", err)
		}
		fmt.Printf("Created test file: %s\n", outputFile)

		// Remove the extracted tests from the original test file
		if err := removeExtractedTests(testFile, matchingTests, fset); err != nil {
			return fmt.Errorf("failed to update original test file: %w", err)
		}
	}

	return nil
}

func writeTestsToFile(filename string, tests []TestFunction, fset *token.FileSet) error {
	if len(tests) == 0 {
		return nil
	}

	var decls []ast.Decl

	// Collect all imports needed
	allImports := tests[0].Imports
	usedPackages := make(map[string]bool)
	usedPackages["testing"] = true

	for _, test := range tests {
		ast.Inspect(test.FuncDecl, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if ident, ok := x.X.(*ast.Ident); ok {
					usedPackages[ident.Name] = true
				}
			}

			return true
		})
	}

	// Add import declarations
	var usedImports []*ast.ImportSpec
	for _, imp := range allImports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		if importPath == "testing" || usedPackages[pkgName] {
			usedImports = append(usedImports, imp)
		}
	}

	if len(usedImports) > 0 {
		importDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(usedImports)),
		}
		for i, imp := range usedImports {
			importDecl.Specs[i] = imp
		}
		decls = append(decls, importDecl)
	}

	// Add all test functions
	for _, test := range tests {
		if test.Comments != nil {
			test.FuncDecl.Doc = test.Comments
		}
		decls = append(decls, test.FuncDecl)
	}

	// Create an AST file
	astFile := &ast.File{
		Name:  &ast.Ident{Name: tests[0].Package},
		Decls: decls,
	}

	// Format and write to file
	if err := formatAndWriteFile(filename, astFile, fset); err != nil {
		return err
	}

	return nil
}

func findUsedImports(fn *ast.FuncDecl, allImports []*ast.ImportSpec) []*ast.ImportSpec {
	usedPackages := make(map[string]bool)

	// Walk through the function body to find used packages
	ast.Inspect(fn, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if ident, ok := x.X.(*ast.Ident); ok {
				usedPackages[ident.Name] = true
			}
		case *ast.CallExpr:
			if ident, ok := x.Fun.(*ast.Ident); ok {
				usedPackages[ident.Name] = true
			}
		case *ast.Ident:
			if x.Obj == nil && x.Name != "" {
				usedPackages[x.Name] = true
			}
		}

		return true
	})

	// For test functions, always include "testing"
	if strings.HasPrefix(fn.Name.Name, "Test") || strings.HasPrefix(fn.Name.Name, "Benchmark") {
		usedPackages["testing"] = true
	}

	// Filter imports to only include used ones
	var result []*ast.ImportSpec
	for _, imp := range allImports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		if importPath == "testing" && usedPackages["testing"] {
			result = append(result, imp)
		} else if usedPackages[pkgName] {
			result = append(result, imp)
		} else if strings.Contains(importPath, "testify/assert") && usedPackages["assert"] {
			result = append(result, imp)
		} else if strings.Contains(importPath, "testify/require") && usedPackages["require"] {
			result = append(result, imp)
		} else if strings.Contains(importPath, "testify/suite") && usedPackages["suite"] {
			result = append(result, imp)
		}
	}

	return result
}

func findUsedImportsInDecls(decls []ast.Decl, allImports []*ast.ImportSpec) []*ast.ImportSpec {
	usedPackages := make(map[string]bool)

	// Check for test functions
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
				if ident, ok := x.X.(*ast.Ident); ok {
					usedPackages[ident.Name] = true
				}
			case *ast.CallExpr:
				if ident, ok := x.Fun.(*ast.Ident); ok {
					usedPackages[ident.Name] = true
				}
			case *ast.Ident:
				if x.Obj == nil && x.Name != "" {
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

		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		if importPath == "testing" && usedPackages["testing"] {
			result = append(result, imp)
		} else if usedPackages[pkgName] {
			result = append(result, imp)
		} else if strings.Contains(importPath, "testify") && usedPackages["assert"] {
			result = append(result, imp)
		} else if strings.Contains(importPath, "testify") && usedPackages["require"] {
			result = append(result, imp)
		} else if strings.Contains(importPath, "testify") && usedPackages["suite"] {
			result = append(result, imp)
		}
	}

	return result
}

func functionNameToSnakeCase(name string) string {
	// Handle common abbreviations
	commonAbbreviations := getCommonAbbreviations()
	for _, abbr := range commonAbbreviations {
		if strings.ToUpper(name) == abbr {
			return strings.ToLower(name)
		}
	}

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
			i += length - 1

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
		return "func"
	}

	// Remove leading underscore if present
	return strings.TrimLeft(resultStr, "_")
}

func testNameToSnakeCase(name string) string {
	if !strings.HasPrefix(name, "Test") {
		return strings.ToLower(name)
	}

	name = strings.TrimPrefix(name, "Test")
	name = strings.TrimLeft(name, "_")

	if name == "" {
		return "test"
	}

	// Check if the entire name is a common abbreviation
	commonAbbreviations := getCommonAbbreviations()
	for _, abbr := range commonAbbreviations {
		if strings.ToUpper(name) == abbr {
			return strings.ToLower(name)
		}
	}

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
			i += length - 1

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

		// Check if it's a word boundary
		atWordBoundary := i+len(abbr) == len(runes) ||
			(i+len(abbr) < len(runes) && unicode.IsUpper(runes[i+len(abbr)]))

		if atWordBoundary {
			return abbr, len(abbr)
		}
	}

	return "", 0
}

func shouldAddUnderscore(runes []rune, i int, result []rune) bool {
	if i == 0 || !unicode.IsUpper(runes[i]) {
		return false
	}

	if len(result) == 0 || result[len(result)-1] == '_' {
		return false
	}

	// Uppercase followed by lowercase
	if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
		return true
	}

	// Lowercase followed by uppercase
	if i > 0 && unicode.IsLower(runes[i-1]) {
		return true
	}

	return false
}
