package splitter

import (
	"errors"
	"go/ast"
)

var ErrTypeCast = errors.New("failed to cast to GenDecl")

type PublicFunction struct {
	Name               string
	FuncDecl           *ast.FuncDecl
	Comments           *ast.CommentGroup
	StandaloneComments []*ast.CommentGroup
	Imports            []*ast.ImportSpec
	Package            string
}

type PublicDeclaration struct {
	GenDecl  *ast.GenDecl
	Comments *ast.CommentGroup
	Package  string
	Imports  []*ast.ImportSpec
}

type TestFunction struct {
	Name               string
	FuncDecl           *ast.FuncDecl
	Comments           *ast.CommentGroup
	StandaloneComments []*ast.CommentGroup
	Imports            []*ast.ImportSpec
	Package            string
}