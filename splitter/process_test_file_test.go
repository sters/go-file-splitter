package splitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessTestFile(t *testing.T) {
	tmpDir := t.TempDir()

	testContent := `package example

import "testing"

func TestFirst(t *testing.T) {
	t.Log("First test")
}

func TestSecond(t *testing.T) {
	t.Log("Second test")
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

	// The original file is now preserved even if it only contains extracted tests
	// because the new logic removes extracted tests from the original file
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		// This is OK - file can be deleted if no remaining content
	} else {
		// File exists - verify it doesn't contain the extracted tests
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read original file: %v", err)
		}
		if strings.Contains(string(content), "func TestFirst") {
			t.Error("Original file should not contain TestFirst after extraction")
		}
		if strings.Contains(string(content), "func TestSecond") {
			t.Error("Original file should not contain TestSecond after extraction")
		}
	}

	expectedFiles := []string{
		filepath.Join(tmpDir, "first_test.go"),
		filepath.Join(tmpDir, "second_test.go"),
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

		if strings.HasSuffix(expectedFile, "first_test.go") {
			if !strings.Contains(string(content), "TestFirst") {
				t.Errorf("File %s should contain TestFirst function", expectedFile)
			}
		} else if strings.HasSuffix(expectedFile, "second_test.go") {
			if !strings.Contains(string(content), "TestSecond") {
				t.Errorf("File %s should contain TestSecond function", expectedFile)
			}
		}
	}
}
