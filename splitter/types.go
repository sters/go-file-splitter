package splitter

import (
	"errors"
	"go/ast"
)

var ErrTypeCast = errors.New("failed to cast to GenDecl")

type MethodStrategy string

const (
	MethodStrategySeparate   MethodStrategy = "separate"
	MethodStrategyWithStruct MethodStrategy = "with-struct"
)

type PublicFunction struct {
	Name               string
	FuncDecl           *ast.FuncDecl
	Comments           *ast.CommentGroup
	StandaloneComments []*ast.CommentGroup
	InlineComments     []*ast.CommentGroup // Comments inside the function body
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
	InlineComments     []*ast.CommentGroup // Comments inside the function body
	Imports            []*ast.ImportSpec
	Package            string
}

type PublicMethod struct {
	Name               string
	ReceiverType       string // The type name of the receiver
	FuncDecl           *ast.FuncDecl
	Comments           *ast.CommentGroup
	StandaloneComments []*ast.CommentGroup
	InlineComments     []*ast.CommentGroup
	Imports            []*ast.ImportSpec
	Package            string
}
