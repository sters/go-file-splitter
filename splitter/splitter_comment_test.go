package splitter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommentHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string][]string // function name -> expected comments
	}{
		{
			name: "comments before and after function",
			input: `package test

import "testing"

// Comment before TestA
// Another comment before TestA
func TestA(t *testing.T) {
	// Internal comment A
	t.Log("A")
}
// Comment after TestA - stays in original

// Comment before TestB
func TestB(t *testing.T) {
	// Internal comment B
	t.Log("B")
}
// Comment after TestB - stays in original
`,
			expected: map[string][]string{
				"TestA": {
					"// Comment before TestA",
					"// Another comment before TestA",
				},
				"TestB": {
					"// Comment before TestB",
				},
			},
		},
		{
			name: "comments between functions",
			input: `package test

import "testing"

func TestFirst(t *testing.T) {
	t.Log("First")
}

// This comment is between functions
// Should go with TestSecond

func TestSecond(t *testing.T) {
	t.Log("Second")
}
`,
			expected: map[string][]string{
				"TestFirst": {},
				"TestSecond": {
					"// This comment is between functions",
					"// Should go with TestSecond",
				},
			},
		},
		{
			name: "helper function preservation",
			input: `package test

import "testing"

func TestMain(t *testing.T) {
	t.Log("Main")
}

// Helper function comment
func helperFunc() {
	// Internal helper comment
}

// End of file comment
`,
			expected: map[string][]string{
				"TestMain": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test_file_test.go")

			// Write test input
			err := os.WriteFile(testFile, []byte(tt.input), 0o644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Parse the file
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			// Extract test functions
			tests, _ := extractTestFunctions(node)

			// Check comments for each test function
			for _, test := range tests {
				expectedComments, exists := tt.expected[test.Name]
				if !exists {
					t.Errorf("Unexpected test function extracted: %s", test.Name)

					continue
				}

				// Collect all comments for this test
				var actualComments []string

				// Add doc comments
				if test.Comments != nil {
					for _, comment := range test.Comments.List {
						actualComments = append(actualComments, comment.Text)
					}
				}

				// Add standalone comments
				for _, cg := range test.StandaloneComments {
					for _, comment := range cg.List {
						actualComments = append(actualComments, comment.Text)
					}
				}

				// Compare expected vs actual
				if len(expectedComments) != len(actualComments) {
					t.Errorf("Test %s: expected %d comments, got %d\nExpected: %v\nActual: %v",
						test.Name, len(expectedComments), len(actualComments),
						expectedComments, actualComments)

					continue
				}

				// Check each comment
				for i, expected := range expectedComments {
					if i >= len(actualComments) {
						break
					}
					if expected != actualComments[i] {
						t.Errorf("Test %s: comment %d mismatch\nExpected: %s\nActual: %s",
							test.Name, i, expected, actualComments[i])
					}
				}
			}
		})
	}
}

func TestIsTestSpecificComment(t *testing.T) {
	testCases := []struct {
		name         string
		code         string
		funcName     string
		commentText  string
		shouldBelong bool
	}{
		{
			name: "comment before function",
			code: `package test
// This comment
func TestA(t *testing.T) {}
`,
			funcName:     "TestA",
			commentText:  "// This comment",
			shouldBelong: true,
		},
		{
			name: "comment far after function",
			code: `package test
func TestA(t *testing.T) {}

// This comment
`,
			funcName:     "TestA",
			commentText:  "// This comment",
			shouldBelong: false, // Comments after functions stay in original
		},
		{
			name: "trailing comment immediately after function",
			code: `package test
func TestA(t *testing.T) {}
// Trailing comment
`,
			funcName:     "TestA",
			commentText:  "// Trailing comment",
			shouldBelong: false, // All comments after functions stay in original
		},
		{
			name: "comment inside function",
			code: `package test
func TestA(t *testing.T) {
	// Internal comment
}
`,
			funcName:     "TestA",
			commentText:  "// Internal comment",
			shouldBelong: false, // Internal comments are handled automatically
		},
		{
			name: "comment closer to next function",
			code: `package test
func TestA(t *testing.T) {}

// Far comment


func TestB(t *testing.T) {}
`,
			funcName:     "TestA",
			commentText:  "// Far comment",
			shouldBelong: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "", tc.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			// Find the target function
			var targetFunc *ast.FuncDecl
			for _, decl := range node.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == tc.funcName {
					targetFunc = fn

					break
				}
			}

			if targetFunc == nil {
				t.Fatalf("Function %s not found", tc.funcName)
			}

			// Find the comment group containing the target comment
			var targetComment *ast.CommentGroup
			for _, cg := range node.Comments {
				for _, c := range cg.List {
					if strings.TrimSpace(c.Text) == strings.TrimSpace(tc.commentText) {
						targetComment = cg

						break
					}
				}
				if targetComment != nil {
					break
				}
			}

			if targetComment == nil {
				t.Fatalf("Comment '%s' not found", tc.commentText)
			}

			// Test the function
			result := isTestSpecificComment(targetComment, targetFunc, node.Decls)
			if result != tc.shouldBelong {
				t.Errorf("Expected isTestSpecificComment = %v, got %v", tc.shouldBelong, result)
			}
		})
	}
}

func TestFilterOrphanedComments(t *testing.T) {
	code := `package test

import "testing"

// Comment before TestExtracted
func TestExtracted(t *testing.T) {
	// Internal comment
}
// Comment after TestExtracted

// Comment before helper
func helperFunc() {}
// Comment after helper
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Simulate extraction of TestExtracted
	extractedNames := map[string]bool{
		"TestExtracted": true,
	}

	// Filter orphaned comments
	filtered := filterOrphanedComments(node, extractedNames)

	// Count comments that should remain (helper function comments)
	remainingComments := 0
	for _, cg := range filtered {
		for _, c := range cg.List {
			comment := strings.TrimSpace(c.Text)
			// Comments related to TestExtracted should be removed
			if strings.Contains(comment, "TestExtracted") || strings.Contains(comment, "Internal") {
				t.Errorf("Comment should have been filtered: %s", comment)
			}
			// Comments related to helper should remain
			if strings.Contains(comment, "helper") {
				remainingComments++
			}
		}
	}

	if remainingComments != 2 {
		t.Errorf("Expected 2 helper comments to remain, got %d", remainingComments)
	}
}
