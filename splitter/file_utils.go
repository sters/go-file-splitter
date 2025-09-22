package splitter

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func findGoFiles(directory string) ([]string, error) {
	var goFiles []string

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			goFiles = append(goFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return goFiles, nil
}

func findTestFiles(directory string) ([]string, error) {
	var testFiles []string

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, "_test.go") {
			testFiles = append(testFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return testFiles, nil
}

func findCorrespondingTestFile(filename string, _ string) string {
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	base = strings.TrimSuffix(base, ".go")
	testFile := filepath.Join(dir, base+"_test.go")

	if _, err := os.Stat(testFile); err == nil {
		return testFile
	}

	return ""
}
