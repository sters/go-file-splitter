package splitter

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"strings"
)

func writePublicFunction(filename string, fn PublicFunction, fset *token.FileSet) error {
	var decls []ast.Decl

	// Find which imports are actually used
	usedImports := findUsedImports(fn.FuncDecl, fn.Imports)

	// Add import declarations if there are any used imports
	if len(usedImports) > 0 {
		importDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(usedImports)),
		}
		for i, imp := range usedImports {
			importDecl.Specs[i] = imp
		}
		decls = append(decls, importDecl)
	}

	// Add the function with its comments
	if fn.Comments != nil {
		fn.FuncDecl.Doc = fn.Comments
	}
	decls = append(decls, fn.FuncDecl)

	// Create an AST file
	astFile := &ast.File{
		Name:     &ast.Ident{Name: fn.Package},
		Decls:    decls,
		Comments: fn.StandaloneComments,
	}

	// Format and write to file
	if err := formatAndWriteFile(filename, astFile, fset); err != nil {
		return err
	}

	return nil
}

func writeTestFunction(filename string, test TestFunction, fset *token.FileSet) error {
	var decls []ast.Decl

	// Find which imports are actually used
	usedImports := findUsedImports(test.FuncDecl, test.Imports)

	// Add import declarations if there are any used imports
	if len(usedImports) > 0 {
		importDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(usedImports)),
		}
		for i, imp := range usedImports {
			importDecl.Specs[i] = imp
		}
		decls = append(decls, importDecl)
	}

	// Add the test function with its comments
	if test.Comments != nil {
		test.FuncDecl.Doc = test.Comments
	}
	decls = append(decls, test.FuncDecl)

	// Create an AST file
	astFile := &ast.File{
		Name:     &ast.Ident{Name: test.Package},
		Decls:    decls,
		Comments: test.StandaloneComments,
	}

	// Format and write to file
	if err := formatAndWriteFile(filename, astFile, fset); err != nil {
		return err
	}

	return nil
}

func writeCommonFile(filename string, decls []PublicDeclaration, pkgName string, imports []*ast.ImportSpec, fset *token.FileSet) error {
	var astDecls []ast.Decl

	// Collect all used imports from declarations
	usedPackages := make(map[string]bool)
	for _, decl := range decls {
		ast.Inspect(decl.GenDecl, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if ident, ok := x.X.(*ast.Ident); ok {
					usedPackages[ident.Name] = true
				}
			}

			return true
		})
	}

	// Filter and add imports
	var usedImports []*ast.ImportSpec
	for _, imp := range imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		var pkgNameFromImport string
		if imp.Name != nil {
			pkgNameFromImport = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgNameFromImport = parts[len(parts)-1]
		}

		if usedPackages[pkgNameFromImport] {
			usedImports = append(usedImports, imp)
		}
	}

	if len(usedImports) > 0 {
		importDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(usedImports)),
		}
		for i, imp := range usedImports {
			importDecl.Specs[i] = imp
		}
		astDecls = append(astDecls, importDecl)
	}

	// Add all public declarations
	for _, decl := range decls {
		astDecls = append(astDecls, decl.GenDecl)
	}

	// Create an AST file
	astFile := &ast.File{
		Name:  &ast.Ident{Name: pkgName},
		Decls: astDecls,
	}

	// Format and write to file
	if err := formatAndWriteFile(filename, astFile, fset); err != nil {
		return err
	}

	return nil
}

func writeTestsToFile(filename string, tests []TestFunction, fset *token.FileSet) error {
	if len(tests) == 0 {
		return nil
	}

	var decls []ast.Decl

	// Collect all imports needed
	allImports := tests[0].Imports
	usedPackages := make(map[string]bool)
	usedPackages["testing"] = true

	for _, test := range tests {
		ast.Inspect(test.FuncDecl, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if ident, ok := x.X.(*ast.Ident); ok {
					usedPackages[ident.Name] = true
				}
			}

			return true
		})
	}

	// Add import declarations
	var usedImports []*ast.ImportSpec
	for _, imp := range allImports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		if importPath == "testing" || usedPackages[pkgName] {
			usedImports = append(usedImports, imp)
		}
	}

	if len(usedImports) > 0 {
		importDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, len(usedImports)),
		}
		for i, imp := range usedImports {
			importDecl.Specs[i] = imp
		}
		decls = append(decls, importDecl)
	}

	// Add all test functions
	for _, test := range tests {
		if test.Comments != nil {
			test.FuncDecl.Doc = test.Comments
		}
		decls = append(decls, test.FuncDecl)
	}

	// Create an AST file
	astFile := &ast.File{
		Name:  &ast.Ident{Name: tests[0].Package},
		Decls: decls,
	}

	// Format and write to file
	if err := formatAndWriteFile(filename, astFile, fset); err != nil {
		return err
	}

	return nil
}

func formatAndWriteFile(filename string, astFile *ast.File, fset *token.FileSet) error {
	var buf strings.Builder
	if err := format.Node(&buf, fset, astFile); err != nil {
		return fmt.Errorf("failed to format code: %w", err)
	}

	if err := os.WriteFile(filename, []byte(buf.String()), 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}