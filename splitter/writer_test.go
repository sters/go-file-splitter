package splitter

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePublicMethod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a method AST
	method := PublicMethod{
		Name:         "GetName",
		ReceiverType: "User",
		FuncDecl: &ast.FuncDecl{
			Recv: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{{Name: "u"}},
						Type:  &ast.Ident{Name: "User"},
					},
				},
			},
			Name: &ast.Ident{Name: "GetName"},
			Type: &ast.FuncType{
				Params: &ast.FieldList{},
				Results: &ast.FieldList{
					List: []*ast.Field{
						{Type: &ast.Ident{Name: "string"}},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.SelectorExpr{
								X:   &ast.Ident{Name: "u"},
								Sel: &ast.Ident{Name: "name"},
							},
						},
					},
				},
			},
		},
		Package: "user",
	}

	outputFile := filepath.Join(tmpDir, "user_get_name.go")
	fset := token.NewFileSet()
	if err := writePublicMethod(outputFile, method, fset); err != nil {
		t.Fatalf("writePublicMethod failed: %v", err)
	}

	// Check file was created
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	expectedContents := []string{
		"package user",
		"func (u User) GetName() string",
		"return u.name",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Output should contain %q", expected)
		}
	}
}

func TestWriteMethodsWithStructs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create methods grouped by type
	methods := []PublicMethod{
		{
			Name:         "Method1",
			ReceiverType: "MyType",
			FuncDecl: &ast.FuncDecl{
				Recv: &ast.FieldList{
					List: []*ast.Field{{
						Names: []*ast.Ident{{Name: "m"}},
						Type:  &ast.Ident{Name: "MyType"},
					}},
				},
				Name: &ast.Ident{Name: "Method1"},
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
				},
				Body: &ast.BlockStmt{},
			},
			Package: "test",
		},
		{
			Name:         "Method2",
			ReceiverType: "MyType",
			FuncDecl: &ast.FuncDecl{
				Recv: &ast.FieldList{
					List: []*ast.Field{{
						Names: []*ast.Ident{{Name: "m"}},
						Type:  &ast.StarExpr{X: &ast.Ident{Name: "MyType"}},
					}},
				},
				Name: &ast.Ident{Name: "Method2"},
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
				},
				Body: &ast.BlockStmt{},
			},
			Package: "test",
		},
	}

	// Create a simple type declaration
	typeDecls := []*ast.GenDecl{
		{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: &ast.Ident{Name: "MyType"},
					Type: &ast.StructType{
						Fields: &ast.FieldList{
							List: []*ast.Field{
								{
									Names: []*ast.Ident{{Name: "field"}},
									Type:  &ast.Ident{Name: "string"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create public declarations from type declarations
	publicDecls := make([]PublicDeclaration, 0, len(typeDecls))
	for _, decl := range typeDecls {
		publicDecls = append(publicDecls, PublicDeclaration{
			GenDecl: decl,
			Package: "test",
		})
	}

	fset := token.NewFileSet()
	if err := writeMethodsWithStructs(tmpDir, publicDecls, methods, "test", nil, fset); err != nil {
		t.Fatalf("writeMethodsWithStructs failed: %v", err)
	}

	// Check file was created - should be my_type.go based on the type name
	outputFile := filepath.Join(tmpDir, "my_type.go")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	expectedContents := []string{
		"package test",
		"type MyType struct",
		"func (m MyType) Method1()",
		"func (m *MyType) Method2()",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Output should contain %q", expected)
		}
	}
}

func TestWriteTypeWithMethods(t *testing.T) {
	tmpDir := t.TempDir()

	// Create type declaration
	typeDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: &ast.Ident{Name: "MyType"},
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: []*ast.Field{
							{
								Names: []*ast.Ident{{Name: "name"}},
								Type:  &ast.Ident{Name: "string"},
							},
						},
					},
				},
			},
		},
	}

	// Create methods for the type
	methods := []PublicMethod{
		{
			Name:         "GetName",
			ReceiverType: "MyType",
			FuncDecl: &ast.FuncDecl{
				Recv: &ast.FieldList{
					List: []*ast.Field{{
						Names: []*ast.Ident{{Name: "m"}},
						Type:  &ast.Ident{Name: "MyType"},
					}},
				},
				Name: &ast.Ident{Name: "GetName"},
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
					Results: &ast.FieldList{
						List: []*ast.Field{
							{Type: &ast.Ident{Name: "string"}},
						},
					},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ReturnStmt{
							Results: []ast.Expr{
								&ast.SelectorExpr{
									X:   &ast.Ident{Name: "m"},
									Sel: &ast.Ident{Name: "name"},
								},
							},
						},
					},
				},
			},
			Package: "test",
		},
	}

	outputFile := filepath.Join(tmpDir, "my_type.go")
	fset := token.NewFileSet()
	if err := writeTypeWithMethods(outputFile, typeDecl, methods, "test", nil, fset); err != nil {
		t.Fatalf("writeTypeWithMethods failed: %v", err)
	}

	// Check file was created
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	expectedContents := []string{
		"package test",
		"type MyType struct",
		"name string",
		"func (m MyType) GetName() string",
		"return m.name",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Output should contain %q", expected)
		}
	}
}

func TestFormatAndWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

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
