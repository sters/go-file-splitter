package splitter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestFindUsedImports(t *testing.T) {
	tests := []struct {
		name              string
		testCode          string
		expectedImports   []string
		unexpectedImports []string
	}{
		{
			name: "test with fmt and strings packages",
			testCode: `package main

import (
	"fmt"
	"strings"
	"bytes"
	"testing"
)

func TestExample(t *testing.T) {
	fmt.Println("test")
	if strings.HasPrefix("hello", "h") {
		t.Log("has prefix")
	}
}`,
			expectedImports:   []string{"fmt", "strings", "testing"},
			unexpectedImports: []string{"bytes"},
		},
		{
			name: "test with only testing package",
			testCode: `package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestSimple(t *testing.T) {
	t.Log("simple test")
	t.Error("error")
}`,
			expectedImports:   []string{"testing"},
			unexpectedImports: []string{"fmt", "strings"},
		},
		{
			name: "test with aliased imports",
			testCode: `package main

import (
	"fmt"
	str "strings"
	"testing"
)

func TestAliased(t *testing.T) {
	str.ToLower("TEST")
	t.Log("done")
}`,
			expectedImports:   []string{"strings", "testing"},
			unexpectedImports: []string{"fmt"},
		},
		{
			name: "test with testify assert",
			testCode: `package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"fmt"
)

func TestWithAssert(t *testing.T) {
	assert.Equal(t, 1, 1)
}`,
			expectedImports:   []string{"testing", "github.com/stretchr/testify/assert"},
			unexpectedImports: []string{"fmt", "github.com/stretchr/testify/require"},
		},
		{
			name: "test with multiple selector expressions",
			testCode: `package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"fmt"
)

func TestMultiplePkgs(t *testing.T) {
	dir := filepath.Join("a", "b")
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
	filepath.Base(dir)
}`,
			expectedImports:   []string{"os", "path/filepath", "testing"},
			unexpectedImports: []string{"strings", "fmt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the test code
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "test.go", tt.testCode, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse test code: %v", err)
			}

			// Find the test function
			var testFunc *ast.FuncDecl
			for _, decl := range node.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name[0:4] == "Test" {
					testFunc = fn

					break
				}
			}

			if testFunc == nil {
				t.Fatal("Test function not found")
			}

			// Get the used imports
			usedImports := findUsedImports(testFunc, node.Imports)

			// Check expected imports are present
			for _, expected := range tt.expectedImports {
				found := false
				for _, imp := range usedImports {
					impPath := imp.Path.Value
					// Remove quotes
					impPath = impPath[1 : len(impPath)-1]

					// Check if this is the expected import
					if impPath == expected {
						found = true

						break
					}

					// For aliased imports, check by alias name
					if imp.Name != nil && imp.Name.Name == expected {
						found = true

						break
					}
				}

				if !found {
					t.Errorf("Expected import %q not found in used imports", expected)
				}
			}

			// Check unexpected imports are NOT present
			for _, unexpected := range tt.unexpectedImports {
				found := false
				for _, imp := range usedImports {
					impPath := imp.Path.Value
					// Remove quotes
					impPath = impPath[1 : len(impPath)-1]

					if impPath == unexpected {
						found = true

						break
					}
				}

				if found {
					t.Errorf("Unexpected import %q found in used imports", unexpected)
				}
			}
		})
	}
}
