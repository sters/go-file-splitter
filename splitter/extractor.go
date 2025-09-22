package splitter

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"
)

func extractPublicFunctions(node *ast.File) []PublicFunction {
	publicFuncs := make([]PublicFunction, 0, len(node.Decls))

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
		var inlineComments []*ast.CommentGroup
		for _, cg := range node.Comments {
			if cg == fn.Doc {
				continue
			}
			// Check if comment is inside the function body
			if fn.Body != nil && cg.Pos() >= fn.Body.Lbrace && cg.End() <= fn.Body.Rbrace {
				inlineComments = append(inlineComments, cg)
			} else if isFunctionSpecificComment(cg, fn, node.Decls) {
				standaloneComments = append(standaloneComments, cg)
			}
		}

		publicFunc := PublicFunction{
			Name:               fn.Name.Name,
			FuncDecl:           fn,
			Comments:           fn.Doc,
			StandaloneComments: standaloneComments,
			InlineComments:     inlineComments,
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

		// Check if this declaration contains any public const/var/type
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
			case *ast.TypeSpec:
				// Check if the type is public
				if unicode.IsUpper(rune(s.Name.Name[0])) {
					hasPublic = true
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
	tests := make([]TestFunction, 0, len(node.Decls))

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
		var inlineComments []*ast.CommentGroup
		for _, cg := range node.Comments {
			if cg == fn.Doc {
				continue
			}
			// Check if comment is inside the function body
			if fn.Body != nil && cg.Pos() >= fn.Body.Lbrace && cg.End() <= fn.Body.Rbrace {
				inlineComments = append(inlineComments, cg)
			} else if isFunctionSpecificComment(cg, fn, node.Decls) {
				standaloneComments = append(standaloneComments, cg)
			}
		}

		test := TestFunction{
			Name:               fn.Name.Name,
			FuncDecl:           fn,
			Comments:           fn.Doc,
			StandaloneComments: standaloneComments,
			InlineComments:     inlineComments,
			Imports:            node.Imports,
			Package:            node.Name.Name,
		}
		tests = append(tests, test)
	}

	return tests
}

func extractPublicMethods(node *ast.File) []PublicMethod {
	publicMethods := make([]PublicMethod, 0, len(node.Decls))

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil {
			continue
		}

		// Check if method is public (starts with uppercase)
		if !unicode.IsUpper(rune(fn.Name.Name[0])) {
			continue
		}

		// Extract receiver type name
		receiverType := getReceiverTypeName(fn.Recv)
		if receiverType == "" {
			continue
		}

		var standaloneComments []*ast.CommentGroup
		var inlineComments []*ast.CommentGroup
		for _, cg := range node.Comments {
			if cg == fn.Doc {
				continue
			}
			// Check if comment is inside the function body
			if fn.Body != nil && cg.Pos() >= fn.Body.Lbrace && cg.End() <= fn.Body.Rbrace {
				inlineComments = append(inlineComments, cg)
			} else if isFunctionSpecificComment(cg, fn, node.Decls) {
				standaloneComments = append(standaloneComments, cg)
			}
		}

		publicMethod := PublicMethod{
			Name:               fn.Name.Name,
			ReceiverType:       receiverType,
			FuncDecl:           fn,
			Comments:           fn.Doc,
			StandaloneComments: standaloneComments,
			InlineComments:     inlineComments,
			Imports:            node.Imports,
			Package:            node.Name.Name,
		}
		publicMethods = append(publicMethods, publicMethod)
	}

	return publicMethods
}

func getReceiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	field := recv.List[0]
	if field.Type == nil {
		return ""
	}

	switch t := field.Type.(type) {
	case *ast.Ident:
		// Simple type: func (r Receiver)
		return t.Name
	case *ast.StarExpr:
		// Pointer type: func (r *Receiver)
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}

	return ""
}
