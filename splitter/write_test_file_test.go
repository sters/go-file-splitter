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

func TestWriteTestFile(t *testing.T) {
	tmpDir := t.TempDir()

	src := `package example
import "testing"
func TestExample(t *testing.T) {
	t.Log("test")
}`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	// Extract test function
	var testFunc *ast.FuncDecl
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "TestExample" {
			testFunc = fn

			break
		}
	}

	if testFunc == nil {
		t.Fatal("Failed to find TestExample function")
	}

	test := TestFunction{
		Name:     "TestExample",
		FuncDecl: testFunc,
		Imports:  node.Imports,
		Package:  "example",
	}

	outputFile := filepath.Join(tmpDir, "example_test.go")
	err = writeTestFile(outputFile, test, fset)
	if err != nil {
		t.Fatalf("writeTestFile failed: %v", err)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "package example") {
		t.Error("Output should contain package declaration")
	}
	if !strings.Contains(contentStr, "TestExample") {
		t.Error("Output should contain test function")
	}
	if !strings.Contains(contentStr, "testing") {
		t.Error("Output should contain testing import")
	}
}
