package splitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessTestFileWithComments(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a test file with comments
	testFile := filepath.Join(tempDir, "example_test.go")
	testContent := `package example

import (
	"testing"
)

// TestFirst is a test function that verifies the first functionality.
// It checks various edge cases and ensures proper behavior.
func TestFirst(t *testing.T) {
	t.Log("First test")
}

// TestSecond validates the secondary features.
// This is a multi-line comment that explains
// the purpose and behavior of this test.
func TestSecond(t *testing.T) {
	t.Log("Second test")
}

// helperFunction is a helper that should remain in the file
func helperFunction() {
	// Do something
}

// TestThird performs validation on third component.
func TestThird(t *testing.T) {
	t.Log("Third test")
}
`

	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process the test file
	if err := processTestFile(testFile); err != nil {
		t.Fatalf("Failed to process test file: %v", err)
	}

	// Check that the split files were created with comments
	expectedFiles := map[string]string{
		"first_test.go":  "// TestFirst is a test function that verifies the first functionality.\n// It checks various edge cases and ensures proper behavior.",
		"second_test.go": "// TestSecond validates the secondary features.\n// This is a multi-line comment that explains\n// the purpose and behavior of this test.",
		"third_test.go":  "// TestThird performs validation on third component.",
	}

	for filename, expectedComment := range expectedFiles {
		filePath := filepath.Join(tempDir, filename)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", filename)

			continue
		}

		// Read the file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read %s: %v", filename, err)

			continue
		}

		// Check if the comment is present in the file
		if !strings.Contains(string(content), expectedComment) {
			t.Errorf("File %s does not contain expected comment.\nExpected comment:\n%s\nActual content:\n%s",
				filename, expectedComment, string(content))
		}

		// Also verify the test function is present
		if !strings.Contains(string(content), "t.Log(") {
			t.Errorf("File %s does not contain the test function body", filename)
		}
	}

	// Check that the original file still contains the helper function
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		return // File was deleted, which is fine if there was no remaining content
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}

	// Should contain the helper function and its comment
	if !strings.Contains(string(content), "helperFunction") {
		t.Error("Original file should contain the helper function")
	}
	if !strings.Contains(string(content), "// helperFunction is a helper that should remain in the file") {
		t.Error("Original file should contain the helper function's comment")
	}

	// Should NOT contain the extracted test functions
	if strings.Contains(string(content), "func TestFirst") {
		t.Errorf("Original file should not contain TestFirst. Content:\n%s", string(content))
	}
	if strings.Contains(string(content), "func TestSecond") {
		t.Error("Original file should not contain TestSecond")
	}
	if strings.Contains(string(content), "func TestThird") {
		t.Error("Original file should not contain TestThird")
	}
}

func TestProcessTestFileCommentsWithInlineComments(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a test file with various comment styles
	testFile := filepath.Join(tempDir, "inline_test.go")
	testContent := `package example

import (
	"testing"
)

// TestWithDocComment has a proper doc comment
func TestWithDocComment(t *testing.T) {
	// This is an inline comment inside the function
	t.Log("Test with doc comment")
}

func TestWithoutComment(t *testing.T) {
	t.Log("Test without comment")
}

// This comment is not attached to any function

// TestAnotherWithComment has its own comment
func TestAnotherWithComment(t *testing.T) {
	t.Log("Another test")
}
`

	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process the test file
	if err := processTestFile(testFile); err != nil {
		t.Fatalf("Failed to process test file: %v", err)
	}

	// Check with_doc_comment_test.go
	withDocCommentFile := filepath.Join(tempDir, "with_doc_comment_test.go")
	content, err := os.ReadFile(withDocCommentFile)
	if err != nil {
		t.Fatalf("Failed to read test_with_doc_comment_test.go: %v", err)
	}

	if !strings.Contains(string(content), "// TestWithDocComment has a proper doc comment") {
		t.Error("test_with_doc_comment_test.go should contain its doc comment")
	}

	// Check without_comment_test.go
	withoutCommentFile := filepath.Join(tempDir, "without_comment_test.go")
	content, err = os.ReadFile(withoutCommentFile)
	if err != nil {
		t.Fatalf("Failed to read test_without_comment_test.go: %v", err)
	}

	// Should not have a doc comment line since the original didn't have one
	if strings.Count(string(content), "// TestWithoutComment") > 0 {
		t.Error("test_without_comment_test.go should not have a doc comment")
	}

	// Check another_with_comment_test.go
	anotherWithCommentFile := filepath.Join(tempDir, "another_with_comment_test.go")
	content, err = os.ReadFile(anotherWithCommentFile)
	if err != nil {
		t.Fatalf("Failed to read test_another_with_comment_test.go: %v", err)
	}

	if !strings.Contains(string(content), "// TestAnotherWithComment has its own comment") {
		t.Error("test_another_with_comment_test.go should contain its doc comment")
	}
}
