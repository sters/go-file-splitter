package splitter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestFindUsedImportsInDecls(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		expectedImports []string
	}{
		{
			name: "helper functions with fmt usage",
			code: `package main

import (
	"fmt"
	"strings"
	"testing"
)

func helperFunction() {
	fmt.Println("helper")
}

func TestSomething(t *testing.T) {
	t.Log("test")
}`,
			expectedImports: []string{"fmt", "testing"},
		},
		{
			name: "only helper functions no test",
			code: `package main

import (
	"fmt"
	"strings"
)

func helperFunction() {
	fmt.Println("helper")
}

func anotherHelper() {
	strings.Contains("test", "t")
}`,
			expectedImports: []string{"fmt", "strings"},
		},
		{
			name: "test with unused imports",
			code: `package main

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestOnlyUsesFmt(t *testing.T) {
	fmt.Println("test")
}`,
			expectedImports: []string{"fmt", "testing"},
		},
		{
			name: "benchmark function",
			code: `package main

import (
	"fmt"
	"testing"
)

func BenchmarkSomething(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fmt.Sprintf("test")
	}
}`,
			expectedImports: []string{"fmt", "testing"},
		},
		{
			name: "example function",
			code: `package main

import (
	"fmt"
	"testing"
)

func ExampleFunction() {
	fmt.Println("example")
}`,
			expectedImports: []string{"fmt", "testing"},
		},
		{
			name: "test with testify",
			code: `package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithTestify(t *testing.T) {
	assert.Equal(t, 1, 1)
}

func TestWithUnusedRequire(t *testing.T) {
	assert.NotNil(t, "test")
}`,
			expectedImports: []string{"testing", "github.com/stretchr/testify/assert"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			// Extract only non-import declarations
			var decls []ast.Decl
			for _, decl := range node.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
					continue
				}
				decls = append(decls, decl)
			}

			usedImports := findUsedImportsInDecls(decls, node.Imports)

			// Check if the number of imports matches
			if len(usedImports) != len(tt.expectedImports) {
				t.Errorf("Expected %d imports, got %d", len(tt.expectedImports), len(usedImports))
				for _, imp := range usedImports {
					t.Logf("Got import: %s", imp.Path.Value)
				}

				return
			}

			// Check if all expected imports are present
			for _, expected := range tt.expectedImports {
				found := false
				for _, imp := range usedImports {
					path := imp.Path.Value
					// Remove quotes
					if len(path) >= 2 && path[0] == '"' && path[len(path)-1] == '"' {
						path = path[1 : len(path)-1]
					}
					if path == expected {
						found = true

						break
					}
				}
				if !found {
					t.Errorf("Expected import %q not found", expected)
				}
			}
		})
	}
}
