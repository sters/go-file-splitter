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
	Name     string
	FuncDecl *ast.FuncDecl
	Imports  []*ast.ImportSpec
	Package  string
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
		fmt.Printf("Preserved original: %s (contains non-split tests or helper functions)\n", filename)
	}

	return nil
}

func extractTestFunctions(node *ast.File) ([]TestFunction, bool) {
	var tests []TestFunction
	hasRemainingContent := false

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fn.Name.Name, "Test") && fn.Recv == nil {
				// Check if the character after "Test" (and any underscores) is uppercase
				nameAfterTest := strings.TrimPrefix(fn.Name.Name, "Test")
				nameAfterTest = strings.TrimLeft(nameAfterTest, "_")

				// Skip if empty or starts with lowercase (e.g., Test_foo)
				if len(nameAfterTest) == 0 || unicode.IsLower(rune(nameAfterTest[0])) {
					hasRemainingContent = true
					continue
				}

				test := TestFunction{
					Name:     fn.Name.Name,
					FuncDecl: fn,
					Imports:  node.Imports,
					Package:  node.Name.Name,
				}
				tests = append(tests, test)
			} else if fn.Recv == nil {
				// Non-test functions (helper functions) should be preserved
				hasRemainingContent = true
			}
		} else if _, ok := decl.(*ast.GenDecl); ok {
			// Type declarations, constants, variables should be preserved
			hasRemainingContent = true
		}
	}

	return tests, hasRemainingContent
}

func writeTestFile(filename string, test TestFunction, fset *token.FileSet) error {
	// Build declarations: imports first, then the test function
	var decls []ast.Decl

	// Add import declarations if there are any imports
	if len(test.Imports) > 0 {
		decls = append(decls, &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(test.Imports)),
		})
		// Copy import specs
		for i, imp := range test.Imports {
			genDecl, ok := decls[0].(*ast.GenDecl)
			if !ok {
				return ErrTypeCast
			}
			genDecl.Specs[i] = imp
		}
	}

	// Add the test function
	decls = append(decls, test.FuncDecl)

	// Create an AST file with the test function and imports
	astFile := &ast.File{
		Name:  &ast.Ident{Name: test.Package},
		Decls: decls,
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

	result := make([]rune, 0, len(name))
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			if i+1 < len(name) && unicode.IsLower(rune(name[i+1])) {
				result = append(result, '_')
			} else if i > 0 && unicode.IsLower(rune(name[i-1])) {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(r))
	}

	resultStr := string(result)
	if resultStr == "" {
		return "test"
	}

	return resultStr
}
