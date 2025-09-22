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

func Test_Uppercase(t *testing.T) {
	// should be extracted (starts with uppercase after underscore)
}

func Test___MultiUnderscore(t *testing.T) {
	// should be extracted (starts with uppercase after underscores)
}

func Test_lowercase(t *testing.T) {
	// should NOT be extracted (starts with lowercase after underscore)
}

func Test__doubleLowercase(t *testing.T) {
	// should NOT be extracted (starts with lowercase after underscores)
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

	tests, hasRemainingContent := extractTestFunctions(node)

	if len(tests) != 4 {
		t.Errorf("Expected 4 test functions, got %d", len(tests))
		for _, test := range tests {
			t.Logf("Found test: %s", test.Name)
		}
	}

	// Verify that hasRemainingContent is true because of lowercase tests, helper functions, and types
	if !hasRemainingContent {
		t.Error("Expected hasRemainingContent to be true due to lowercase tests, helper functions, and type declarations")
	}

	expectedNames := []string{"TestExample1", "TestExample2", "Test_Uppercase", "Test___MultiUnderscore"}
	for i, expectedName := range expectedNames {
		if i >= len(tests) {
			t.Errorf("Missing expected test: %q", expectedName)

			continue
		}
		if tests[i].Name != expectedName {
			t.Errorf("Expected test name %q at index %d, got %q", expectedName, i, tests[i].Name)
		}
		if tests[i].Package != "example" {
			t.Errorf("Expected package name 'example', got %q", tests[i].Package)
		}
		if tests[i].FuncDecl == nil {
			t.Errorf("Expected FuncDecl to be non-nil for %s", tests[i].Name)
		}
	}

	// Verify that lowercase tests are not included
	for _, test := range tests {
		if test.Name == "Test_lowercase" || test.Name == "Test__doubleLowercase" {
			t.Errorf("Test %q should not have been extracted (starts with lowercase)", test.Name)
		}
	}
}
