package splitter

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"
)

func extractPublicFunctions(node *ast.File) []PublicFunction {
	var publicFuncs []PublicFunction

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}

		// Check if function is public (starts with uppercase)
		if !unicode.IsUpper(rune(fn.Name.Name[0])) {
			continue
		}

		var standaloneComments []*ast.CommentGroup
		for _, cg := range node.Comments {
			if cg == fn.Doc {
				continue
			}
			if isFunctionSpecificComment(cg, fn, node.Decls) {
				standaloneComments = append(standaloneComments, cg)
			}
		}

		publicFunc := PublicFunction{
			Name:               fn.Name.Name,
			FuncDecl:           fn,
			Comments:           fn.Doc,
			StandaloneComments: standaloneComments,
			Imports:            node.Imports,
			Package:            node.Name.Name,
		}
		publicFuncs = append(publicFuncs, publicFunc)
	}

	return publicFuncs
}

func extractPublicDeclarations(node *ast.File) []PublicDeclaration {
	var publicDecls []PublicDeclaration

	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok == token.IMPORT {
			continue
		}

		// Check if this declaration contains any public const/var
		hasPublic := false
		for _, spec := range genDecl.Specs {
			switch s := spec.(type) {
			case *ast.ValueSpec:
				for _, name := range s.Names {
					if unicode.IsUpper(rune(name.Name[0])) {
						hasPublic = true

						break
					}
				}
			}
			if hasPublic {
				break
			}
		}

		if hasPublic {
			publicDecl := PublicDeclaration{
				GenDecl:  genDecl,
				Comments: genDecl.Doc,
				Package:  node.Name.Name,
				Imports:  node.Imports,
			}
			publicDecls = append(publicDecls, publicDecl)
		}
	}

	return publicDecls
}

func extractTestFunctions(node *ast.File) []TestFunction {
	var tests []TestFunction

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}

		if !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}

		// Check if the character after "Test" (and any underscores) is uppercase
		nameAfterTest := strings.TrimPrefix(fn.Name.Name, "Test")
		nameAfterTest = strings.TrimLeft(nameAfterTest, "_")

		// Skip if empty or starts with lowercase
		if len(nameAfterTest) == 0 || unicode.IsLower(rune(nameAfterTest[0])) {
			continue
		}

		var standaloneComments []*ast.CommentGroup
		for _, cg := range node.Comments {
			if cg == fn.Doc {
				continue
			}
			if isFunctionSpecificComment(cg, fn, node.Decls) {
				standaloneComments = append(standaloneComments, cg)
			}
		}

		test := TestFunction{
			Name:               fn.Name.Name,
			FuncDecl:           fn,
			Comments:           fn.Doc,
			StandaloneComments: standaloneComments,
			Imports:            node.Imports,
			Package:            node.Name.Name,
		}
		tests = append(tests, test)
	}

	return tests
}