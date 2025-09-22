package splitter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	testFiles := []string{
		"example_test.go",
		"another_test.go",
		"subdir/nested_test.go",
	}

	nonTestFiles := []string{
		"main.go",
		"helper.go",
		"subdir/util.go",
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0o755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("package test"), 0o644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	for _, file := range nonTestFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("package main"), 0o644); err != nil {
			t.Fatalf("Failed to create non-test file %s: %v", file, err)
		}
	}

	found, err := findTestFiles(tmpDir)
	if err != nil {
		t.Fatalf("findTestFiles failed: %v", err)
	}

	if len(found) != len(testFiles) {
		t.Errorf("Expected %d test files, found %d", len(testFiles), len(found))
	}

	for _, expectedFile := range testFiles {
		expectedPath := filepath.Join(tmpDir, expectedFile)
		fileFound := false
		for _, foundFile := range found {
			if foundFile == expectedPath {
				fileFound = true

				break
			}
		}
		if !fileFound {
			t.Errorf("Expected test file %s was not found", expectedFile)
		}
	}
}
