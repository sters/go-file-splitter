package splitter

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func SplitPublicFunctions(directory string, strategy MethodStrategy) error {
	goFiles, err := findGoFiles(directory)
	if err != nil {
		return fmt.Errorf("failed to find go files: %w", err)
	}

	for _, file := range goFiles {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if err := processGoFile(file, strategy); err != nil {
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

func processGoFile(filename string, strategy MethodStrategy) error {
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
	publicMethods := extractPublicMethods(node)

	if len(publicFuncs) == 0 && len(publicDecls) == 0 && len(publicMethods) == 0 {
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

	// Handle methods based on strategy
	if err := writeMethodsAndDeclarations(strategy, outputDir, publicDecls, publicMethods, node.Name.Name, node.Imports, fset); err != nil {
		return err
	}

	// Update original file to keep only private content
	if err := updateOriginalFile(filename, publicFuncs, publicDecls, publicMethods, fset); err != nil {
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

func updateOriginalFile(filename string, extractedFuncs []PublicFunction, extractedDecls []PublicDeclaration, extractedMethods []PublicMethod, fset *token.FileSet) error {
	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Create extraction maps
	extractedFuncNames, extractedDeclPtrs, extractedMethodKeys := buildExtractionMaps(extractedFuncs, extractedDecls, extractedMethods)

	// Filter declarations
	newDecls, hasRemainingContent := filterDeclarations(node.Decls, extractedFuncNames, extractedDeclPtrs, extractedMethodKeys)

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

	// Filter comments to remove those belonging to extracted functions
	// We need to collect comment text that should be removed
	removedCommentTexts := make(map[string]bool)

	// Collect comments to remove
	addFunctionComments(&removedCommentTexts, extractedFuncs)
	addDeclarationComments(&removedCommentTexts, extractedDecls)
	addMethodComments(&removedCommentTexts, extractedMethods)

	// Keep only comment groups that don't contain removed comment texts
	var remainingComments []*ast.CommentGroup
	for _, cg := range node.Comments {
		shouldKeep := true
		for _, c := range cg.List {
			if removedCommentTexts[c.Text] {
				shouldKeep = false

				break
			}
		}
		if shouldKeep {
			remainingComments = append(remainingComments, cg)
		}
	}
	node.Comments = remainingComments

	// Format and write back
	if err := formatAndWriteFile(filename, node, fset); err != nil {
		return err
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

	// Filter comments to remove those belonging to extracted tests
	removedCommentTexts := make(map[string]bool)

	// Collect comment texts from extracted tests
	for _, test := range extractedTests {
		// Use the original FuncDecl from the extracted test
		if test.FuncDecl != nil {
			// Remove doc comments
			if test.FuncDecl.Doc != nil {
				for _, c := range test.FuncDecl.Doc.List {
					removedCommentTexts[c.Text] = true
				}
			}
			// Remove inline comments collected during extraction
			for _, cg := range test.InlineComments {
				for _, c := range cg.List {
					removedCommentTexts[c.Text] = true
				}
			}
			// Remove standalone comments
			for _, cg := range test.StandaloneComments {
				for _, c := range cg.List {
					removedCommentTexts[c.Text] = true
				}
			}
		}
	}

	// Keep only comment groups that don't contain removed comment texts
	var remainingComments []*ast.CommentGroup
	for _, cg := range node.Comments {
		shouldKeep := true
		for _, c := range cg.List {
			if removedCommentTexts[c.Text] {
				shouldKeep = false

				break
			}
		}
		if shouldKeep {
			remainingComments = append(remainingComments, cg)
		}
	}
	node.Comments = remainingComments

	// Format and write back
	if err := formatAndWriteFile(filename, node, fset); err != nil {
		return err
	}

	fmt.Printf("Preserved original: %s (contains non-split tests or helper functions)\n", filename)

	return nil
}

// writeMethodsAndDeclarations handles writing methods and declarations based on strategy.
func writeMethodsAndDeclarations(strategy MethodStrategy, outputDir string, publicDecls []PublicDeclaration, publicMethods []PublicMethod, packageName string, imports []*ast.ImportSpec, fset *token.FileSet) error {
	if strategy == MethodStrategyWithStruct {
		return writeMethodsWithStructs(outputDir, publicDecls, publicMethods, packageName, imports, fset)
	}

	// Strategy: separate - Write methods to individual files
	if err := writeSeparateMethods(outputDir, publicMethods, fset); err != nil {
		return err
	}

	// Write public const/var/type declarations to common.go
	if len(publicDecls) > 0 {
		commonFile := filepath.Join(outputDir, "common.go")
		if err := writeCommonFile(commonFile, publicDecls, packageName, imports, fset); err != nil {
			return fmt.Errorf("failed to write common.go: %w", err)
		}
		fmt.Printf("Created: %s\n", commonFile)
	}

	return nil
}

// writeSeparateMethods writes each method to its own file.
func writeSeparateMethods(outputDir string, publicMethods []PublicMethod, fset *token.FileSet) error {
	for _, method := range publicMethods {
		snakeCaseName := methodNameToSnakeCase(method.ReceiverType, method.Name)
		outputFileName := snakeCaseName + ".go"
		outputFile := filepath.Join(outputDir, outputFileName)

		if err := writePublicMethod(outputFile, method, fset); err != nil {
			return fmt.Errorf("failed to write method file %s: %w", outputFile, err)
		}
		fmt.Printf("Created: %s\n", outputFile)
	}

	return nil
}

// Helper functions for updateOriginalFile to reduce complexity

func buildExtractionMaps(extractedFuncs []PublicFunction, extractedDecls []PublicDeclaration, extractedMethods []PublicMethod) (map[string]bool, map[*ast.GenDecl]bool, map[string]bool) {
	extractedFuncNames := make(map[string]bool)
	for _, fn := range extractedFuncs {
		extractedFuncNames[fn.Name] = true
	}

	extractedDeclPtrs := make(map[*ast.GenDecl]bool)
	for _, decl := range extractedDecls {
		extractedDeclPtrs[decl.GenDecl] = true
	}

	extractedMethodKeys := make(map[string]bool)
	for _, method := range extractedMethods {
		key := method.ReceiverType + "." + method.Name
		extractedMethodKeys[key] = true
	}

	return extractedFuncNames, extractedDeclPtrs, extractedMethodKeys
}

func filterDeclarations(decls []ast.Decl, extractedFuncNames map[string]bool, extractedDeclPtrs map[*ast.GenDecl]bool, extractedMethodKeys map[string]bool) ([]ast.Decl, bool) {
	var newDecls []ast.Decl
	hasRemainingContent := false

	for _, decl := range decls {
		if shouldKeepDeclaration(decl, extractedFuncNames, extractedDeclPtrs, extractedMethodKeys) {
			newDecls = append(newDecls, decl)
			hasRemainingContent = true
		}
	}

	return newDecls, hasRemainingContent
}

func shouldKeepDeclaration(decl ast.Decl, extractedFuncNames map[string]bool, extractedDeclPtrs map[*ast.GenDecl]bool, extractedMethodKeys map[string]bool) bool {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		return shouldKeepFunction(d, extractedFuncNames, extractedMethodKeys)
	case *ast.GenDecl:
		return shouldKeepGenDecl(d, extractedDeclPtrs)
	default:
		return false
	}
}

func shouldKeepFunction(d *ast.FuncDecl, extractedFuncNames map[string]bool, extractedMethodKeys map[string]bool) bool {
	// Check if this is a method that was extracted
	if d.Recv != nil {
		receiverType := getReceiverTypeName(d.Recv)
		if receiverType != "" {
			key := receiverType + "." + d.Name.Name
			if extractedMethodKeys[key] {
				return false
			}
		}
	}

	// Keep if not in extracted functions
	return !extractedFuncNames[d.Name.Name]
}

func shouldKeepGenDecl(d *ast.GenDecl, extractedDeclPtrs map[*ast.GenDecl]bool) bool {
	if d.Tok == token.IMPORT {
		return false // We'll re-add imports later if needed
	}

	// Keep private declarations
	if extractedDeclPtrs[d] {
		return false
	}

	// Check if this declaration has any private members
	return hasPrivateMembers(d)
}

func hasPrivateMembers(d *ast.GenDecl) bool {
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			for _, name := range s.Names {
				if !unicode.IsUpper(rune(name.Name[0])) {
					return true
				}
			}
		case *ast.TypeSpec:
			if !unicode.IsUpper(rune(s.Name.Name[0])) {
				return true
			}
		}
	}

	return false
}

func addFunctionComments(removedCommentTexts *map[string]bool, extractedFuncs []PublicFunction) {
	for _, fn := range extractedFuncs {
		if fn.FuncDecl == nil {
			continue
		}

		if fn.FuncDecl.Doc != nil {
			for _, c := range fn.FuncDecl.Doc.List {
				(*removedCommentTexts)[c.Text] = true
			}
		}

		for _, cg := range fn.InlineComments {
			for _, c := range cg.List {
				(*removedCommentTexts)[c.Text] = true
			}
		}

		for _, cg := range fn.StandaloneComments {
			for _, c := range cg.List {
				(*removedCommentTexts)[c.Text] = true
			}
		}
	}
}

func addDeclarationComments(removedCommentTexts *map[string]bool, extractedDecls []PublicDeclaration) {
	for _, decl := range extractedDecls {
		if decl.Comments != nil {
			for _, c := range decl.Comments.List {
				(*removedCommentTexts)[c.Text] = true
			}
		}
	}
}

func addMethodComments(removedCommentTexts *map[string]bool, extractedMethods []PublicMethod) {
	for _, method := range extractedMethods {
		if method.FuncDecl == nil {
			continue
		}

		if method.FuncDecl.Doc != nil {
			for _, c := range method.FuncDecl.Doc.List {
				(*removedCommentTexts)[c.Text] = true
			}
		}

		for _, cg := range method.InlineComments {
			for _, c := range cg.List {
				(*removedCommentTexts)[c.Text] = true
			}
		}

		for _, cg := range method.StandaloneComments {
			for _, c := range cg.List {
				(*removedCommentTexts)[c.Text] = true
			}
		}
	}
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
			var inlineComments []*ast.CommentGroup
			for _, cg := range node.Comments {
				if cg == fn.Doc {
					continue
				}
				// Check if comment is inside the function body
				if fn.Body != nil && cg.Pos() >= fn.Body.Lbrace && cg.End() <= fn.Body.Rbrace {
					inlineComments = append(inlineComments, cg)
				} else if isFunctionSpecificComment(cg, fn, node.Decls) {
					standaloneComments = append(standaloneComments, cg)
				}
			}

			test := TestFunction{
				Name:               fn.Name.Name,
				FuncDecl:           fn,
				Comments:           fn.Doc,
				StandaloneComments: standaloneComments,
				InlineComments:     inlineComments,
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
