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
