package splitter

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func writePublicFunction(filename string, fn PublicFunction, fset *token.FileSet) error {
	return writeFunctionGeneric(filename, fn.FuncDecl, fn.Comments, fn.StandaloneComments, fn.InlineComments, fn.Imports, fn.Package, fset)
}

func writeTestFunction(filename string, test TestFunction, fset *token.FileSet) error {
	return writeFunctionGeneric(filename, test.FuncDecl, test.Comments, test.StandaloneComments, test.InlineComments, test.Imports, test.Package, fset)
}

// writeFunctionGeneric is a generic function to write a function (either public or test) to a file.
func writeFunctionGeneric(filename string, funcDecl *ast.FuncDecl, comments *ast.CommentGroup, standaloneComments, inlineComments []*ast.CommentGroup, imports []*ast.ImportSpec, packageName string, fset *token.FileSet) error {
	var decls []ast.Decl

	// Find which imports are actually used
	usedImports := findUsedImports(funcDecl, imports)

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
	if comments != nil {
		funcDecl.Doc = comments
	}
	decls = append(decls, funcDecl)

	// Combine all comments: doc, standalone, and inline
	var allComments []*ast.CommentGroup
	if comments != nil {
		allComments = append(allComments, comments)
	}
	allComments = append(allComments, standaloneComments...)
	allComments = append(allComments, inlineComments...)

	// Create an AST file
	astFile := &ast.File{
		Name:     &ast.Ident{Name: packageName},
		Decls:    decls,
		Comments: allComments,
	}

	// Format and write to file
	return formatAndWriteFile(filename, astFile, fset)
}

func writeCommonFile(filename string, decls []PublicDeclaration, pkgName string, imports []*ast.ImportSpec, fset *token.FileSet) error {
	astDecls := make([]ast.Decl, 0, len(decls)+1)

	// Collect all used imports from declarations
	usedPackages := make(map[string]bool)
	for _, decl := range decls {
		ast.Inspect(decl.GenDecl, func(n ast.Node) bool {
			if x, ok := n.(*ast.SelectorExpr); ok {
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

	decls := make([]ast.Decl, 0, len(tests)+1)

	// Collect all imports needed
	allImports := tests[0].Imports
	usedPackages := make(map[string]bool)
	usedPackages["testing"] = true

	for _, test := range tests {
		ast.Inspect(test.FuncDecl, func(n ast.Node) bool {
			if x, ok := n.(*ast.SelectorExpr); ok {
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

func writePublicMethod(filename string, method PublicMethod, fset *token.FileSet) error {
	// Build the declarations
	var decls []ast.Decl

	// Find required imports
	usedPackages := findUsedPackages(method.FuncDecl)
	var usedImports []*ast.ImportSpec
	for _, imp := range method.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		if usedPackages[pkgName] {
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

	// Add the method with its comment
	if method.Comments != nil {
		method.FuncDecl.Doc = method.Comments
	}
	decls = append(decls, method.FuncDecl)

	// Create an AST file
	astFile := &ast.File{
		Name:  &ast.Ident{Name: method.Package},
		Decls: decls,
	}

	// Add inline comments
	if len(method.InlineComments) > 0 {
		astFile.Comments = method.InlineComments
	}

	// Add standalone comments if present
	if len(method.StandaloneComments) > 0 {
		astFile.Comments = append(method.StandaloneComments, astFile.Comments...)
	}

	// Format and write to file
	if err := formatAndWriteFile(filename, astFile, fset); err != nil {
		return err
	}

	return nil
}

func writeMethodsWithStructs(outputDir string, publicDecls []PublicDeclaration, publicMethods []PublicMethod, packageName string, imports []*ast.ImportSpec, fset *token.FileSet) error {
	// Group methods by their receiver type
	methodsByType := make(map[string][]PublicMethod)
	for _, method := range publicMethods {
		methodsByType[method.ReceiverType] = append(methodsByType[method.ReceiverType], method)
	}

	// Collect type declarations
	typeDecls := make(map[string]*ast.GenDecl)
	otherDecls := []PublicDeclaration{}

	for _, decl := range publicDecls {
		hasType := false
		for _, spec := range decl.GenDecl.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				typeDecls[ts.Name.Name] = decl.GenDecl
				hasType = true
			}
		}
		if !hasType {
			otherDecls = append(otherDecls, decl)
		}
	}

	// Write each type with its methods to a separate file
	for typeName, typeDecl := range typeDecls {
		methods := methodsByType[typeName]

		snakeCaseName := functionNameToSnakeCase(typeName)
		outputFileName := snakeCaseName + ".go"
		outputFile := filepath.Join(outputDir, outputFileName)

		if err := writeTypeWithMethods(outputFile, typeDecl, methods, packageName, imports, fset); err != nil {
			return fmt.Errorf("failed to write type file %s: %w", outputFile, err)
		}
		fmt.Printf("Created: %s (with %d methods)\n", outputFile, len(methods))
	}

	// Write types without methods and other declarations to common.go
	if len(otherDecls) > 0 {
		// Add types that don't have methods
		for typeName, typeDecl := range typeDecls {
			if _, hasMethods := methodsByType[typeName]; !hasMethods {
				otherDecls = append(otherDecls, PublicDeclaration{
					GenDecl: typeDecl,
					Package: packageName,
					Imports: imports,
				})
			}
		}

		if len(otherDecls) > 0 {
			commonFile := filepath.Join(outputDir, "common.go")
			if err := writeCommonFile(commonFile, otherDecls, packageName, imports, fset); err != nil {
				return fmt.Errorf("failed to write common.go: %w", err)
			}
			fmt.Printf("Created: %s\n", commonFile)
		}
	}

	// Write orphaned methods (methods whose types aren't found)
	for typeName, methods := range methodsByType {
		if _, found := typeDecls[typeName]; !found {
			// Write each orphaned method separately
			for _, method := range methods {
				snakeCaseName := methodNameToSnakeCase(method.ReceiverType, method.Name)
				outputFileName := snakeCaseName + ".go"
				outputFile := filepath.Join(outputDir, outputFileName)

				if err := writePublicMethod(outputFile, method, fset); err != nil {
					return fmt.Errorf("failed to write orphaned method file %s: %w", outputFile, err)
				}
				fmt.Printf("Created: %s (orphaned method)\n", outputFile)
			}
		}
	}

	return nil
}

func writeTypeWithMethods(filename string, typeDecl *ast.GenDecl, methods []PublicMethod, packageName string, imports []*ast.ImportSpec, fset *token.FileSet) error {
	// Build the declarations
	decls := make([]ast.Decl, 0, len(methods)+2)

	// Find all used packages
	usedPackages := make(map[string]bool)

	// Check type declaration for used packages
	ast.Inspect(typeDecl, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				usedPackages[ident.Name] = true
			}
		}

		return true
	})

	// Check methods for used packages
	for _, method := range methods {
		for pkg := range findUsedPackages(method.FuncDecl) {
			usedPackages[pkg] = true
		}
	}

	// Add used imports
	var usedImports []*ast.ImportSpec
	for _, imp := range imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		if usedPackages[pkgName] {
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

	// Add the type declaration
	decls = append(decls, typeDecl)

	// Add all methods
	for _, method := range methods {
		if method.Comments != nil {
			method.FuncDecl.Doc = method.Comments
		}
		decls = append(decls, method.FuncDecl)
	}

	// Create an AST file
	astFile := &ast.File{
		Name:  &ast.Ident{Name: packageName},
		Decls: decls,
	}

	// Add comments from methods
	var allComments []*ast.CommentGroup
	for _, method := range methods {
		allComments = append(allComments, method.StandaloneComments...)
		allComments = append(allComments, method.InlineComments...)
	}
	if len(allComments) > 0 {
		astFile.Comments = allComments
	}

	// Format and write to file
	if err := formatAndWriteFile(filename, astFile, fset); err != nil {
		return err
	}

	return nil
}
