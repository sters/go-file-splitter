package splitter

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestExtractTestFunctions(t *testing.T) {
	src := `
package example

import "testing"

func TestExample1(t *testing.T) {
	// test code
}

func TestExample2(t *testing.T) {
	// test code
}

func helperFunction() {
	// not a test
}

func BenchmarkExample(b *testing.B) {
	// benchmark, not a test
}

type MyType struct{}

func (m MyType) TestMethodTest(t *testing.T) {
	// method with Test prefix, should be ignored
}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	tests := extractTestFunctions(node)

	if len(tests) != 2 {
		t.Errorf("Expected 2 test functions, got %d", len(tests))
	}

	expectedNames := []string{"TestExample1", "TestExample2"}
	for i, test := range tests {
		if test.Name != expectedNames[i] {
			t.Errorf("Expected test name %q, got %q", expectedNames[i], test.Name)
		}
		if test.Package != "example" {
			t.Errorf("Expected package name 'example', got %q", test.Package)
		}
		if test.FuncDecl == nil {
			t.Errorf("Expected FuncDecl to be non-nil for %s", test.Name)
		}
	}
}
