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

	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Error("Original main_test.go should have been deleted")
	}
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("Original sub_test.go should have been deleted")
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
