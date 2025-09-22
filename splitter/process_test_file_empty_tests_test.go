package splitter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessTestFileEmptyTests(t *testing.T) {
	tmpDir := t.TempDir()

	nonTestContent := `package example

func helperFunction() {
	// not a test
}
`

	nonTestFile := filepath.Join(tmpDir, "helper_test.go")
	if err := os.WriteFile(nonTestFile, []byte(nonTestContent), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	err := processTestFile(nonTestFile)
	if err != nil {
		t.Fatalf("processTestFile failed: %v", err)
	}

	if _, err := os.Stat(nonTestFile); os.IsNotExist(err) {
		t.Error("File with no tests should not be deleted")
	}
}
