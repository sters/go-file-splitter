package splitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessTestFileWithPreservedContent(t *testing.T) {
	tmpDir := t.TempDir()

	testContent := `package example

import "testing"

// Helper function should be preserved
func setupTest(t *testing.T) {
	t.Log("Setup")
}

// Type declaration should be preserved
type TestHelper struct {
	value int
}

// Constant should be preserved
const testConstant = "test"

// Variable should be preserved
var testVariable = "test"

// Regular test - should be extracted
func TestRegular(t *testing.T) {
	t.Log("Regular test")
}

// Test with lowercase after underscore - should be preserved
func Test_lowercase(t *testing.T) {
	t.Log("Lowercase test")
}

// Test with multiple underscores and lowercase - should be preserved
func Test__anotherLowercase(t *testing.T) {
	t.Log("Another lowercase test")
}

// Test with uppercase after underscore - should be extracted
func Test_Uppercase(t *testing.T) {
	t.Log("Uppercase test")
}

// Benchmark - should be preserved
func BenchmarkSomething(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// benchmark code
	}
}

// Example function - should be preserved
func ExampleFunction() {
	// example code
}
`

	testFile := filepath.Join(tmpDir, "example_test.go")
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := processTestFile(testFile)
	if err != nil {
		t.Fatalf("processTestFile failed: %v", err)
	}

	// Original file should still exist because it has content to preserve
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Original test file should have been preserved")
	}

	// Check original file content
	originalContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}

	// Verify preserved content in original file
	preservedItems := []string{
		"func setupTest",
		"type TestHelper",
		"const testConstant",
		"var testVariable",
		"func Test_lowercase",
		"func Test__anotherLowercase",
		"func BenchmarkSomething",
		"func ExampleFunction",
	}

	for _, item := range preservedItems {
		if !strings.Contains(string(originalContent), item) {
			t.Errorf("Original file should still contain '%s'", item)
		}
	}

	// Verify extracted tests are NOT in original file
	extractedTests := []string{
		"func TestRegular",
		"func Test_Uppercase",
	}

	for _, test := range extractedTests {
		if strings.Contains(string(originalContent), test) {
			t.Errorf("Original file should NOT contain extracted test '%s'", test)
		}
	}

	// Check extracted test files
	expectedFiles := []string{
		filepath.Join(tmpDir, "regular_test.go"),
		filepath.Join(tmpDir, "uppercase_test.go"),
	}

	for _, expectedFile := range expectedFiles {
		_, err := os.Stat(expectedFile)
		if os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", expectedFile)
			continue
		}

		content, err := os.ReadFile(expectedFile)
		if err != nil {
			t.Errorf("Failed to read %s: %v", expectedFile, err)
			continue
		}

		if !strings.Contains(string(content), "package example") {
			t.Errorf("File %s should contain 'package example'", expectedFile)
		}

		if strings.HasSuffix(expectedFile, "regular_test.go") {
			if !strings.Contains(string(content), "TestRegular") {
				t.Errorf("File %s should contain TestRegular function", expectedFile)
			}
		} else if strings.HasSuffix(expectedFile, "uppercase_test.go") {
			if !strings.Contains(string(content), "Test_Uppercase") {
				t.Errorf("File %s should contain Test_Uppercase function", expectedFile)
			}
		}
	}
}