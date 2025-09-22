package splitter

import "testing"

func TestTestNameToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple test name",
			input:    "TestSimple",
			expected: "simple",
		},
		{
			name:     "camel case test name",
			input:    "TestCamelCase",
			expected: "camel_case",
		},
		{
			name:     "multiple uppercase letters",
			input:    "TestHTTPServer",
			expected: "http_server",
		},
		{
			name:     "single word",
			input:    "TestA",
			expected: "a",
		},
		{
			name:     "empty after Test prefix",
			input:    "Test",
			expected: "test",
		},
		{
			name:     "non-test name",
			input:    "NotATest",
			expected: "notatest",
		},
		{
			name:     "complex camel case",
			input:    "TestComplexCamelCaseExample",
			expected: "complex_camel_case_example",
		},
		{
			name:     "test with underscore and uppercase",
			input:    "Test_Uppercase",
			expected: "uppercase",
		},
		{
			name:     "test with multiple underscores and uppercase",
			input:    "Test___MultiUnderscore",
			expected: "multi_underscore",
		},
		{
			name:     "test that starts with uppercase",
			input:    "TestUppercase",
			expected: "uppercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testNameToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("testNameToSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
