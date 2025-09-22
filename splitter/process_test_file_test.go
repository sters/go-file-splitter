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

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("Original test file should have been deleted")
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
