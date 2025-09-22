package splitter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCorrespondingTestFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	mainFile := filepath.Join(tmpDir, "example.go")
	testFile := filepath.Join(tmpDir, "example_test.go")

	if err := os.WriteFile(mainFile, []byte("package test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("package test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test finding corresponding test file
	found := findCorrespondingTestFile(mainFile, "Example")
	if found != testFile {
		t.Errorf("Expected to find %s, got %s", testFile, found)
	}

	// Test when test file doesn't exist
	nonExistent := filepath.Join(tmpDir, "nonexistent.go")
	found = findCorrespondingTestFile(nonExistent, "NonExistent")
	if found != "" {
		t.Errorf("Expected empty string for non-existent test file, got %s", found)
	}
}