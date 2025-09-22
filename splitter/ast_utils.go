package splitter

import (
	"go/ast"
	"go/token"
	"strings"
)

func isFunctionSpecificComment(cg *ast.CommentGroup, fn *ast.FuncDecl, allDecls []ast.Decl) bool {
	// Skip if comment is inside the function body
	if fn.Body != nil && cg.Pos() >= fn.Body.Lbrace && cg.End() <= fn.Body.Rbrace {
		return false
	}

	// Skip if this comment is inside another function body
	for _, otherDecl := range allDecls {
		if otherFn, ok := otherDecl.(*ast.FuncDecl); ok && otherFn != fn {
			if otherFn.Body != nil && cg.Pos() >= otherFn.Body.Lbrace && cg.End() <= otherFn.Body.Rbrace {
				return false
			}
		}
	}

	// Find the function's position in the declarations
	fnIndex := -1
	for i, decl := range allDecls {
		if decl == fn {
			fnIndex = i

			break
		}
	}

	if fnIndex == -1 {
		return false
	}

	// Check comments before the function
	if cg.End() >= fn.Pos() {
		if fn.Body != nil && cg.Pos() > fn.Body.Rbrace {
			return false
		}

		return false
	}

	// Find the previous declaration
	var prevDecl ast.Decl
	prevDeclEnd := token.Pos(0)
	for i := fnIndex - 1; i >= 0; i-- {
		if decl := allDecls[i]; decl != nil {
			prevDecl = decl
			if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Body != nil {
				prevDeclEnd = funcDecl.Body.Rbrace
			} else {
				prevDeclEnd = decl.End()
			}

			break
		}
	}

	// If there's a previous declaration, check which function the comment is closer to
	if prevDecl != nil {
		distToPrevDecl := cg.Pos() - prevDeclEnd
		distToCurrentFunc := fn.Pos() - cg.End()

		// If comment is closer to previous declaration, it belongs to that
		if distToPrevDecl < distToCurrentFunc {
			return false
		}
	}

	// Comment belongs to this function if it's after the previous declaration
	// and reasonably close to the function
	return cg.Pos() > prevDeclEnd && fn.Pos()-cg.End() < token.Pos(50*80)
}

func findUsedImports(fn *ast.FuncDecl, allImports []*ast.ImportSpec) []*ast.ImportSpec {
	usedPackages := make(map[string]bool)

	// Walk through the function body to find used packages
	ast.Inspect(fn, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if ident, ok := x.X.(*ast.Ident); ok {
				usedPackages[ident.Name] = true
			}
		case *ast.CallExpr:
			if ident, ok := x.Fun.(*ast.Ident); ok {
				usedPackages[ident.Name] = true
			}
		case *ast.Ident:
			if x.Obj == nil && x.Name != "" {
				usedPackages[x.Name] = true
			}
		}

		return true
	})

	// For test functions, always include "testing"
	if strings.HasPrefix(fn.Name.Name, "Test") || strings.HasPrefix(fn.Name.Name, "Benchmark") {
		usedPackages["testing"] = true
	}

	// Filter imports to only include used ones
	var result []*ast.ImportSpec
	for _, imp := range allImports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		if usedPackages[pkgName] {
			result = append(result, imp)
		}
	}

	return result
}

func findUsedImportsInDecls(decls []ast.Decl, allImports []*ast.ImportSpec) []*ast.ImportSpec {
	usedPackages := make(map[string]bool)

	// Check for test functions
	for _, decl := range decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fn.Name.Name, "Test") || strings.HasPrefix(fn.Name.Name, "Benchmark") || strings.HasPrefix(fn.Name.Name, "Example") {
				usedPackages["testing"] = true

				break
			}
		}
	}

	// Walk through all declarations to find used packages
	for _, decl := range decls {
		ast.Inspect(decl, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if ident, ok := x.X.(*ast.Ident); ok {
					usedPackages[ident.Name] = true
				}
			case *ast.CallExpr:
				if ident, ok := x.Fun.(*ast.Ident); ok {
					usedPackages[ident.Name] = true
				}
			case *ast.Ident:
				if x.Obj == nil && x.Name != "" {
					usedPackages[x.Name] = true
				}
			}

			return true
		})
	}

	// Filter imports to only include used ones
	var result []*ast.ImportSpec
	for _, imp := range allImports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		if usedPackages[pkgName] {
			result = append(result, imp)
		}
	}

	return result
}

func findUsedPackages(fn *ast.FuncDecl) map[string]bool {
	usedPackages := make(map[string]bool)

	// Walk through the function body and find used packages
	ast.Inspect(fn, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			// Package.Function or Package.Type
			if ident, ok := x.X.(*ast.Ident); ok {
				usedPackages[ident.Name] = true
			}
		case *ast.CallExpr:
			// Check for builtin functions that might need imports
			if ident, ok := x.Fun.(*ast.Ident); ok {
				switch ident.Name {
				case "fmt.Sprintf", "fmt.Printf", "fmt.Println", "fmt.Errorf":
					usedPackages["fmt"] = true
				}
			}
		}

		return true
	})

	return usedPackages
}
