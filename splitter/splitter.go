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

	// Filter comments to remove those belonging to extracted functions
	// We need to collect comment text that should be removed
	removedCommentTexts := make(map[string]bool)

	// Collect comment texts from extracted functions
	for _, fn := range extractedFuncs {
		// Use the original FuncDecl from the extracted function
		if fn.FuncDecl != nil {
			// Remove doc comments
			if fn.FuncDecl.Doc != nil {
				for _, c := range fn.FuncDecl.Doc.List {
					removedCommentTexts[c.Text] = true
				}
			}
			// Remove inline comments collected during extraction
			for _, cg := range fn.InlineComments {
				for _, c := range cg.List {
					removedCommentTexts[c.Text] = true
				}
			}
			// Remove standalone comments
			for _, cg := range fn.StandaloneComments {
				for _, c := range cg.List {
					removedCommentTexts[c.Text] = true
				}
			}
		}
	}

	// Collect comment texts from extracted declarations
	for _, decl := range extractedDecls {
		if decl.Comments != nil {
			for _, c := range decl.Comments.List {
				removedCommentTexts[c.Text] = true
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



