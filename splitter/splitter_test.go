package splitter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractPublicFunctions(t *testing.T) {
	src := `package test

import "fmt"

// PublicFunc is a public function
func PublicFunc() string {
	return "public"
}

// privateFunc is a private function
func privateFunc() string {
	return "private"
}

// PublicMethod is a method (should be ignored)
func (r *Receiver) PublicMethod() {}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	funcs := extractPublicFunctions(node)

	if len(funcs) != 1 {
		t.Errorf("Expected 1 public function, got %d", len(funcs))
	}

	if funcs[0].Name != "PublicFunc" {
		t.Errorf("Expected function name PublicFunc, got %s", funcs[0].Name)
	}
}

func TestExtractPublicDeclarations(t *testing.T) {
	src := `package test

const (
	PublicConst = 1
	privateConst = 2
)

var (
	PublicVar = "public"
	privateVar = "private"
)

type PublicType struct{}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	decls := extractPublicDeclarations(node)

	// Should extract const and var declarations that contain public members
	if len(decls) != 2 {
		t.Errorf("Expected 2 public declarations, got %d", len(decls))
	}
}

func TestExtractTestFunctions(t *testing.T) {
	src := `package test

import "testing"

func TestPublic(t *testing.T) {}
func TestAnother(t *testing.T) {}
func Test_Underscore(t *testing.T) {}
func Test_lowercase(t *testing.T) {} // Should be ignored
func BenchmarkSomething(b *testing.B) {} // Should be ignored
func helperFunc() {} // Should be ignored
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	tests := extractTestFunctions(node)

	if len(tests) != 3 {
		t.Errorf("Expected 3 test functions, got %d", len(tests))
	}

	expectedNames := map[string]bool{
		"TestPublic":      true,
		"TestAnother":     true,
		"Test_Underscore": true,
	}

	for _, test := range tests {
		if !expectedNames[test.Name] {
			t.Errorf("Unexpected test function: %s", test.Name)
		}
	}
}

func TestFunctionNameToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"PublicFunction", "public_function"},
		{"HTTPServer", "http_server"},
		{"GetURL", "get_url"},
		{"ID", "id"},
		{"GetHTTPSURL", "get_https_url"},
		{"SimpleFunc", "simple_func"},
		{"", "func"},
		{"A", "a"},
		{"ABC", "abc"},
		{"XMLParser", "xml_parser"},
	}

	for _, tc := range tests {
		result := functionNameToSnakeCase(tc.input)
		if result != tc.expected {
			t.Errorf("functionNameToSnakeCase(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestTestNameToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"TestPublicFunction", "public_function"},
		{"TestHTTPServer", "http_server"},
		{"Test_Underscore", "underscore"},
		{"TestGetURL", "get_url"},
		{"TestID", "id"},
		{"Test", "test"},
		{"TestA", "a"},
		{"Test_", "test"},
		{"NotTest", "nottest"},
	}

	for _, tc := range tests {
		result := testNameToSnakeCase(tc.input)
		if result != tc.expected {
			t.Errorf("testNameToSnakeCase(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestMatchesAbbreviation(t *testing.T) {
	tests := []struct {
		input    string
		pos      int
		expected string
		length   int
	}{
		{"HTTPServer", 0, "HTTP", 4},
		{"GetURL", 3, "URL", 3},
		{"APIKEY", 0, "API", 3},
		{"NotAbbr", 0, "", 0},
		{"URLParser", 0, "URL", 3},
	}

	for _, tc := range tests {
		runes := []rune(tc.input)
		abbr, length := matchesAbbreviation(runes, tc.pos)
		if abbr != tc.expected || length != tc.length {
			t.Errorf("matchesAbbreviation(%q, %d) = (%q, %d), want (%q, %d)",
				tc.input, tc.pos, abbr, length, tc.expected, tc.length)
		}
	}
}

func TestShouldAddUnderscore(t *testing.T) {
	tests := []struct {
		input    string
		pos      int
		expected bool
	}{
		{"PublicFunc", 6, true},  // Before 'F' in Func
		{"HTTPServer", 4, true},  // Before 'S' in Server
		{"getURL", 3, true},      // Before 'U' in URL
		{"ABC", 1, false},        // All caps
		{"abc", 1, false},        // All lowercase
		{"Public", 0, false},     // First character
	}

	for _, tc := range tests {
		runes := []rune(tc.input)
		result := make([]rune, tc.pos)
		for i := 0; i < tc.pos && i < len(runes); i++ {
			result[i] = runes[i]
		}

		got := shouldAddUnderscore(runes, tc.pos, result)
		if got != tc.expected {
			t.Errorf("shouldAddUnderscore(%q, %d) = %v, want %v",
				tc.input, tc.pos, got, tc.expected)
		}
	}
}

func TestFindUsedImports(t *testing.T) {
	src := `package test

import (
	"fmt"
	"strings"
	"os"
	"testing"
)

func TestExample(t *testing.T) {
	fmt.Println("test")
	strings.ToUpper("test")
}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find the test function
	var testFunc *ast.FuncDecl
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "TestExample" {
			testFunc = fn
			break
		}
	}

	if testFunc == nil {
		t.Fatal("Test function not found")
	}

	usedImports := findUsedImports(testFunc, node.Imports)

	// Should include fmt, strings, and testing (always for test functions)
	expectedCount := 3
	if len(usedImports) != expectedCount {
		t.Errorf("Expected %d used imports, got %d", expectedCount, len(usedImports))
	}

	// Check that os is not included
	for _, imp := range usedImports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path == "os" {
			t.Error("Unused import 'os' should not be included")
		}
	}
}

func TestIsFunctionSpecificComment(t *testing.T) {
	src := `package test

// This comment belongs to the package

// This comment belongs to FirstFunc
func FirstFunc() {}

// This comment is between functions

// This comment belongs to SecondFunc
func SecondFunc() {
	// This is inside the function
}

// This comment is after SecondFunc
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find SecondFunc
	var secondFunc *ast.FuncDecl
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "SecondFunc" {
			secondFunc = fn
			break
		}
	}

	if secondFunc == nil {
		t.Fatal("SecondFunc not found")
	}

	// Test each comment group
	for _, cg := range node.Comments {
		commentText := cg.List[0].Text
		isSpecific := isFunctionSpecificComment(cg, secondFunc, node.Decls)

		// Only the comment "This comment belongs to SecondFunc" should be specific
		shouldBeSpecific := strings.Contains(commentText, "belongs to SecondFunc")
		if isSpecific != shouldBeSpecific {
			t.Errorf("Comment %q: isFunctionSpecificComment = %v, want %v",
				commentText, isSpecific, shouldBeSpecific)
		}
	}
}

func TestSplitPublicFunctions_Integration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "splitter_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test Go file
	testFile := filepath.Join(tmpDir, "example.go")
	testContent := `package example

import "fmt"

const PublicConst = 42

var PublicVar = "test"

func PublicFunc() string {
	return fmt.Sprintf("public")
}

func privateFunc() string {
	return "private"
}
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a corresponding test file
	testTestFile := filepath.Join(tmpDir, "example_test.go")
	testTestContent := `package example

import "testing"

func TestPublicFunc(t *testing.T) {
	if PublicFunc() != "public" {
		t.Error("unexpected result")
	}
}

func TestPrivateFunc(t *testing.T) {
	if privateFunc() != "private" {
		t.Error("unexpected result")
	}
}
`

	if err := os.WriteFile(testTestFile, []byte(testTestContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run SplitPublicFunctions
	if err := SplitPublicFunctions(tmpDir); err != nil {
		t.Fatalf("SplitPublicFunctions failed: %v", err)
	}

	// Check that files were created
	expectedFiles := []string{
		"public_func.go",
		"public_func_test.go",
		"common.go",
	}

	for _, expectedFile := range expectedFiles {
		fullPath := filepath.Join(tmpDir, expectedFile)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", expectedFile)
		}
	}

	// Check that original files were updated
	originalContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Original file should only contain private function
	if !strings.Contains(string(originalContent), "privateFunc") {
		t.Error("Original file should still contain privateFunc")
	}
	if strings.Contains(string(originalContent), "PublicFunc") {
		t.Error("Original file should not contain PublicFunc")
	}
}

func TestSplitTestFunctions_Integration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "splitter_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "example_test.go")
	testContent := `package example

import "testing"

func TestFirst(t *testing.T) {
	t.Log("first")
}

func TestSecond(t *testing.T) {
	t.Log("second")
}

func helperFunc() {
	// Helper function
}
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run SplitTestFunctions
	if err := SplitTestFunctions(tmpDir); err != nil {
		t.Fatalf("SplitTestFunctions failed: %v", err)
	}

	// Check that test files were created
	expectedFiles := []string{
		"first_test.go",
		"second_test.go",
	}

	for _, expectedFile := range expectedFiles {
		fullPath := filepath.Join(tmpDir, expectedFile)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", expectedFile)
		}
	}

	// Check that original file still contains helper function
	originalContent, err := os.ReadFile(testFile)
	if err == nil { // File might be deleted if only tests were present
		if !strings.Contains(string(originalContent), "helperFunc") {
			t.Error("Original file should still contain helperFunc")
		}
	}
}

func TestFindCorrespondingTestFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	mainFile := filepath.Join(tmpDir, "example.go")
	testFile := filepath.Join(tmpDir, "example_test.go")

	if err := os.WriteFile(mainFile, []byte("package test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("package test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test finding corresponding test file
	found := findCorrespondingTestFile(mainFile, "Example")
	if found != testFile {
		t.Errorf("Expected to find %s, got %s", testFile, found)
	}

	// Test when test file doesn't exist
	nonExistent := filepath.Join(tmpDir, "nonexistent.go")
	found = findCorrespondingTestFile(nonExistent, "NonExistent")
	if found != "" {
		t.Errorf("Expected empty string for non-existent test file, got %s", found)
	}
}

func TestFormatAndWriteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fset := token.NewFileSet()

	// Create a simple AST
	astFile := &ast.File{
		Name: &ast.Ident{Name: "test"},
		Decls: []ast.Decl{
			&ast.FuncDecl{
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{},
				},
			},
		},
	}

	outputFile := filepath.Join(tmpDir, "output.go")
	if err := formatAndWriteFile(outputFile, astFile, fset); err != nil {
		t.Fatalf("formatAndWriteFile failed: %v", err)
	}

	// Check file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Check content is valid Go code
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "package test") {
		t.Error("Output file should contain package declaration")
	}
	if !strings.Contains(string(content), "func TestFunc()") {
		t.Error("Output file should contain function declaration")
	}
}