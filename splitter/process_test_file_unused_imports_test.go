package splitter

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessTestFileRemovesUnusedImports(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a test file with multiple imports but only some are used
	testContent := `package example

import (
	"fmt"
	"strings"
	"bytes"
	"testing"
	"os"
	"io"
	"path/filepath"
)

// TestFirst uses fmt and strings
func TestFirst(t *testing.T) {
	fmt.Println("first test")
	if strings.HasPrefix("test", "t") {
		t.Log("prefix found")
	}
}

// TestSecond uses os and filepath
func TestSecond(t *testing.T) {
	path := filepath.Join("a", "b")
	if _, err := os.Stat(path); err != nil {
		t.Error(err)
	}
}

// TestThird uses no extra imports, only testing
func TestThird(t *testing.T) {
	t.Log("simple test")
	t.Run("subtest", func(t *testing.T) {
		t.Log("subtest")
	})
}
`

	testFile := filepath.Join(tmpDir, "example_test.go")
	if err := os.WriteFile(testFile, []byte(testContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Process the test file
	if err := processTestFile(testFile); err != nil {
		t.Fatal(err)
	}

	// Verify the created test files
	testCases := []struct {
		filename          string
		expectedImports   []string
		unexpectedImports []string
	}{
		{
			filename:          "first_test.go",
			expectedImports:   []string{"fmt", "strings", "testing"},
			unexpectedImports: []string{"bytes", "os", "io", "path/filepath"},
		},
		{
			filename:          "second_test.go",
			expectedImports:   []string{"os", "path/filepath", "testing"},
			unexpectedImports: []string{"fmt", "strings", "bytes", "io"},
		},
		{
			filename:          "third_test.go",
			expectedImports:   []string{"testing"},
			unexpectedImports: []string{"fmt", "strings", "bytes", "os", "io", "path/filepath"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			// Read the generated file
			generatedFile := filepath.Join(tmpDir, tc.filename)
			content, err := os.ReadFile(generatedFile)
			if err != nil {
				t.Fatalf("Failed to read generated file %s: %v", tc.filename, err)
			}

			// Parse the generated file
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.filename, content, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse generated file %s: %v", tc.filename, err)
			}

			// Check expected imports are present
			for _, expected := range tc.expectedImports {
				found := false
				for _, imp := range node.Imports {
					impPath := strings.Trim(imp.Path.Value, `"`)
					if impPath == expected {
						found = true

						break
					}
				}
				if !found {
					t.Errorf("Expected import %q not found in %s", expected, tc.filename)
				}
			}

			// Check unexpected imports are NOT present
			for _, unexpected := range tc.unexpectedImports {
				for _, imp := range node.Imports {
					impPath := strings.Trim(imp.Path.Value, `"`)
					if impPath == unexpected {
						t.Errorf("Unexpected import %q found in %s", unexpected, tc.filename)
					}
				}
			}

			// Verify the file compiles
			if _, err := parser.ParseFile(fset, tc.filename, content, 0); err != nil {
				t.Errorf("Generated file %s does not compile: %v", tc.filename, err)
			}
		})
	}
}

func TestProcessTestFileUnusedImportsWithTestify(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a test file with testify imports
	testContent := `package example

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"fmt"
)

// TestWithAssert uses assert
func TestWithAssert(t *testing.T) {
	assert.Equal(t, 1, 1)
	assert.NotNil(t, "test")
}

// TestWithRequire uses require
func TestWithRequire(t *testing.T) {
	require.NoError(t, nil)
	require.True(t, true)
}

// TestNoTestify doesn't use testify
func TestNoTestify(t *testing.T) {
	t.Log("simple test without testify")
}
`

	testFile := filepath.Join(tmpDir, "testify_test.go")
	if err := os.WriteFile(testFile, []byte(testContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Process the test file
	if err := processTestFile(testFile); err != nil {
		t.Fatal(err)
	}

	// Verify the created test files
	testCases := []struct {
		filename      string
		checkContent  string
		shouldContain bool
	}{
		{
			filename:      "with_assert_test.go",
			checkContent:  "github.com/stretchr/testify/assert",
			shouldContain: true,
		},
		{
			filename:      "with_assert_test.go",
			checkContent:  "github.com/stretchr/testify/require",
			shouldContain: false,
		},
		{
			filename:      "with_require_test.go",
			checkContent:  "github.com/stretchr/testify/require",
			shouldContain: true,
		},
		{
			filename:      "with_require_test.go",
			checkContent:  "github.com/stretchr/testify/assert",
			shouldContain: false,
		},
		{
			filename:      "no_testify_test.go",
			checkContent:  "github.com/stretchr/testify",
			shouldContain: false,
		},
		{
			filename:      "no_testify_test.go",
			checkContent:  "fmt",
			shouldContain: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.filename+"_"+tc.checkContent, func(t *testing.T) {
			// Read the generated file
			generatedFile := filepath.Join(tmpDir, tc.filename)
			content, err := os.ReadFile(generatedFile)
			if err != nil {
				t.Fatalf("Failed to read generated file %s: %v", tc.filename, err)
			}

			contentStr := string(content)
			if tc.shouldContain {
				if !strings.Contains(contentStr, tc.checkContent) {
					t.Errorf("Expected %s to contain %q, but it doesn't", tc.filename, tc.checkContent)
				}
			} else {
				if strings.Contains(contentStr, tc.checkContent) {
					t.Errorf("Expected %s NOT to contain %q, but it does", tc.filename, tc.checkContent)
				}
			}
		})
	}
}
