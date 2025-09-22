package splitter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "subpkg")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	testContent1 := `package main
import "testing"
func TestMain(t *testing.T) {
	t.Log("main test")
}`

	testContent2 := `package subpkg
import "testing"
func TestSubPkg(t *testing.T) {
	t.Log("subpkg test")
}`

	file1 := filepath.Join(tmpDir, "main_test.go")
	file2 := filepath.Join(subDir, "sub_test.go")

	if err := os.WriteFile(file1, []byte(testContent1), 0o644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(testContent2), 0o644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	err := SplitTestFiles(tmpDir)
	if err != nil {
		t.Fatalf("SplitTestFiles failed: %v", err)
	}

	// Original files should be deleted since they only contain extracted tests
	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		// Check if file exists and verify it doesn't contain extracted tests
		content, err := os.ReadFile(file1)
		if err != nil {
			t.Fatalf("Failed to read file1: %v", err)
		}
		// The file should be deleted or empty after extraction
		if len(content) > 0 {
			t.Errorf("Original main_test.go should be empty or deleted, but contains: %s", string(content))
		}
	}
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		// Check if file exists and verify it doesn't contain extracted tests
		content, err := os.ReadFile(file2)
		if err != nil {
			t.Fatalf("Failed to read file2: %v", err)
		}
		// The file should be deleted or empty after extraction
		if len(content) > 0 {
			t.Errorf("Original sub_test.go should be empty or deleted, but contains: %s", string(content))
		}
	}

	expectedFiles := []string{
		filepath.Join(tmpDir, "splitted_main_test.go"),
		filepath.Join(subDir, "sub_pkg_test.go"),
	}

	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", expectedFile)
		}
	}
}
