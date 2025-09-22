package splitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitPublicFunctions_Integration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "splitter_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test Go file
	testFile := filepath.Join(tmpDir, "example.go")
	testContent := `package example

import "fmt"

const PublicConst = 42

var PublicVar = "test"

func PublicFunc() string {
	return fmt.Sprintf("public")
}

func privateFunc() string {
	return "private"
}
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a corresponding test file
	testTestFile := filepath.Join(tmpDir, "example_test.go")
	testTestContent := `package example

import "testing"

func TestPublicFunc(t *testing.T) {
	if PublicFunc() != "public" {
		t.Error("unexpected result")
	}
}

func TestPrivateFunc(t *testing.T) {
	if privateFunc() != "private" {
		t.Error("unexpected result")
	}
}
`

	if err := os.WriteFile(testTestFile, []byte(testTestContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run SplitPublicFunctions
	if err := SplitPublicFunctions(tmpDir); err != nil {
		t.Fatalf("SplitPublicFunctions failed: %v", err)
	}

	// Check that files were created
	expectedFiles := []string{
		"public_func.go",
		"public_func_test.go",
		"common.go",
	}

	for _, expectedFile := range expectedFiles {
		fullPath := filepath.Join(tmpDir, expectedFile)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", expectedFile)
		}
	}

	// Check that original files were updated
	originalContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Original file should only contain private function
	if !strings.Contains(string(originalContent), "privateFunc") {
		t.Error("Original file should still contain privateFunc")
	}
	if strings.Contains(string(originalContent), "PublicFunc") {
		t.Error("Original file should not contain PublicFunc")
	}
}

func TestSplitTestFunctions_Integration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "splitter_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "example_test.go")
	testContent := `package example

import "testing"

func TestFirst(t *testing.T) {
	t.Log("first")
}

func TestSecond(t *testing.T) {
	t.Log("second")
}

func helperFunc() {
	// Helper function
}
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run SplitTestFunctions
	if err := SplitTestFunctions(tmpDir); err != nil {
		t.Fatalf("SplitTestFunctions failed: %v", err)
	}

	// Check that test files were created
	expectedFiles := []string{
		"first_test.go",
		"second_test.go",
	}

	for _, expectedFile := range expectedFiles {
		fullPath := filepath.Join(tmpDir, expectedFile)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", expectedFile)
		}
	}

	// Check that original file still contains helper function
	originalContent, err := os.ReadFile(testFile)
	if err == nil { // File might be deleted if only tests were present
		if !strings.Contains(string(originalContent), "helperFunc") {
			t.Error("Original file should still contain helperFunc")
		}
	}
}