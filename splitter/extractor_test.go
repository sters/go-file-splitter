package splitter

import (
	"go/parser"
	"go/token"
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