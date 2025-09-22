package splitter

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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