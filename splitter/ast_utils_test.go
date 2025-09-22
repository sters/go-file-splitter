package splitter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

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

func TestFindUsedPackages(t *testing.T) {
	src := `package test

import (
	"fmt"
	"strings"
	"os"
)

func Example() {
	fmt.Println("test")
	strings.ToUpper("test")
	var _ os.File
}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find the function
	var fn *ast.FuncDecl
	for _, decl := range node.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "Example" {
			fn = f

			break
		}
	}

	if fn == nil {
		t.Fatal("Function not found")
	}

	usedPkgs := findUsedPackages(fn)

	expectedPkgs := map[string]bool{
		"fmt":     true,
		"strings": true,
		"os":      true,
	}

	if len(usedPkgs) != len(expectedPkgs) {
		t.Errorf("Expected %d used packages, got %d", len(expectedPkgs), len(usedPkgs))
	}

	// usedPkgs is a map[string]bool, so iterate over keys
	for pkgName := range usedPkgs {
		if !expectedPkgs[pkgName] {
			t.Errorf("Unexpected package: %s", pkgName)
		}
	}

	for pkgName := range expectedPkgs {
		if !usedPkgs[pkgName] {
			t.Errorf("Missing expected package: %s", pkgName)
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
