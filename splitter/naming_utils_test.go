package splitter

import (
	"testing"
)

func TestFunctionNameToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"PublicFunction", "public_function"},
		{"HTTPServer", "http_server"},
		{"GetURL", "get_url"},
		{"ID", "id"},
		{"GetHTTPSURL", "get_https_url"},
		{"SimpleFunc", "simple_func"},
		{"", "func"},
		{"A", "a"},
		{"ABC", "abc"},
		{"XMLParser", "xml_parser"},
	}

	for _, tc := range tests {
		result := functionNameToSnakeCase(tc.input)
		if result != tc.expected {
			t.Errorf("functionNameToSnakeCase(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestTestNameToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"TestPublicFunction", "public_function"},
		{"TestHTTPServer", "http_server"},
		{"Test_Underscore", "underscore"},
		{"TestGetURL", "get_url"},
		{"TestID", "id"},
		{"Test", "test"},
		{"TestA", "a"},
		{"Test_", "test"},
		{"NotTest", "nottest"},
	}

	for _, tc := range tests {
		result := testNameToSnakeCase(tc.input)
		if result != tc.expected {
			t.Errorf("testNameToSnakeCase(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestMatchesAbbreviation(t *testing.T) {
	tests := []struct {
		input    string
		pos      int
		expected string
		length   int
	}{
		{"HTTPServer", 0, "HTTP", 4},
		{"GetURL", 3, "URL", 3},
		{"APIKEY", 0, "API", 3},
		{"NotAbbr", 0, "", 0},
		{"URLParser", 0, "URL", 3},
	}

	for _, tc := range tests {
		runes := []rune(tc.input)
		abbr, length := matchesAbbreviation(runes, tc.pos)
		if abbr != tc.expected || length != tc.length {
			t.Errorf("matchesAbbreviation(%q, %d) = (%q, %d), want (%q, %d)",
				tc.input, tc.pos, abbr, length, tc.expected, tc.length)
		}
	}
}

func TestMethodNameToSnakeCase(t *testing.T) {
	tests := []struct {
		receiverType string
		methodName   string
		expected     string
	}{
		{"User", "GetName", "user_get_name"},
		{"HTTPClient", "SendRequest", "http_client_send_request"},
		{"DB", "Connect", "db_connect"},
		{"XMLParser", "Parse", "xml_parser_parse"},
		{"MyStruct", "DoSomething", "my_struct_do_something"},
		{"", "OrphanMethod", "func_orphan_method"}, // empty receiver becomes "func"
		{"Service", "", "service_func"},            // empty method becomes "func"
	}

	for _, tc := range tests {
		result := methodNameToSnakeCase(tc.receiverType, tc.methodName)
		if result != tc.expected {
			t.Errorf("methodNameToSnakeCase(%q, %q) = %q, want %q",
				tc.receiverType, tc.methodName, result, tc.expected)
		}
	}
}

func TestShouldAddUnderscore(t *testing.T) {
	tests := []struct {
		input    string
		pos      int
		expected bool
	}{
		{"PublicFunc", 6, true}, // Before 'F' in Func
		{"HTTPServer", 4, true}, // Before 'S' in Server
		{"getURL", 3, true},     // Before 'U' in URL
		{"ABC", 1, false},       // All caps
		{"abc", 1, false},       // All lowercase
		{"Public", 0, false},    // First character
	}

	for _, tc := range tests {
		runes := []rune(tc.input)
		result := make([]rune, tc.pos)
		for i := 0; i < tc.pos && i < len(runes); i++ {
			result[i] = runes[i]
		}

		got := shouldAddUnderscore(runes, tc.pos, result)
		if got != tc.expected {
			t.Errorf("shouldAddUnderscore(%q, %d) = %v, want %v",
				tc.input, tc.pos, got, tc.expected)
		}
	}
}
