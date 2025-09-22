package splitter

import (
	"go/ast"
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

	// Should extract const, var, and type declarations that contain public members
	if len(decls) != 3 {
		t.Errorf("Expected 3 public declarations, got %d", len(decls))
	}
}

func TestExtractPublicMethods(t *testing.T) {
	src := `package test

type MyStruct struct{}
type myPrivateStruct struct{}

// PublicMethod is a public method
func (m MyStruct) PublicMethod() string {
	return "public"
}

// AnotherPublic is another public method
func (m *MyStruct) AnotherPublic() {}

// privateMethod is private
func (m MyStruct) privateMethod() {}

// PublicOnPrivate won't be extracted as receiver is private
func (p myPrivateStruct) PublicOnPrivate() {}

// Regular function
func RegularFunc() {}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	methods := extractPublicMethods(node)

	// PublicOnPrivate is also extracted since the method itself is public
	if len(methods) != 3 {
		t.Errorf("Expected 3 public methods, got %d", len(methods))
	}

	expectedMethods := map[string]string{
		"PublicMethod":    "MyStruct",
		"AnotherPublic":   "MyStruct",
		"PublicOnPrivate": "myPrivateStruct",
	}

	for _, method := range methods {
		if expectedType, ok := expectedMethods[method.Name]; !ok || method.ReceiverType != expectedType {
			t.Errorf("Unexpected method: %s with receiver %s", method.Name, method.ReceiverType)
		}
	}
}

func TestGetReceiverTypeName(t *testing.T) {
	src := `package test

func (m MyStruct) Method1() {}
func (m *MyStruct) Method2() {}
func (m AnotherStruct) Method3() {}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	tests := []struct {
		index    int
		expected string
	}{
		{0, "MyStruct"},
		{1, "MyStruct"},
		{2, "AnotherStruct"},
	}

	for i, test := range tests {
		fn, ok := node.Decls[i].(*ast.FuncDecl)
		if !ok {
			t.Fatalf("Test %d: Expected FuncDecl but got different type", i)
		}
		result := getReceiverTypeName(fn.Recv)
		if result != test.expected {
			t.Errorf("Test %d: Expected %s, got %s", i, test.expected, result)
		}
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
