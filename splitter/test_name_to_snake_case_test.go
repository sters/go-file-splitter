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
			name:     "ID abbreviation",
			input:    "TestID",
			expected: "id",
		},
		{
			name:     "UserID mixed",
			input:    "TestUserID",
			expected: "user_id",
		},
		{
			name:     "IDUser mixed",
			input:    "TestIDUser",
			expected: "id_user",
		},
		{
			name:     "multiple abbreviations",
			input:    "TestHTTPServerAPI",
			expected: "http_server_api",
		},
		{
			name:     "URL abbreviation",
			input:    "TestURL",
			expected: "url",
		},
		{
			name:     "ParseURL mixed",
			input:    "TestParseURL",
			expected: "parse_url",
		},
		{
			name:     "UUID abbreviation",
			input:    "TestUUID",
			expected: "uuid",
		},
		{
			name:     "GenerateUUID mixed",
			input:    "TestGenerateUUID",
			expected: "generate_uuid",
		},
		{
			name:     "JSON abbreviation",
			input:    "TestJSON",
			expected: "json",
		},
		{
			name:     "ParseJSONData mixed",
			input:    "TestParseJSONData",
			expected: "parse_json_data",
		},
		{
			name:     "XML abbreviation",
			input:    "TestXML",
			expected: "xml",
		},
		{
			name:     "HTTP abbreviation",
			input:    "TestHTTP",
			expected: "http",
		},
		{
			name:     "HTTPSConnection mixed",
			input:    "TestHTTPSConnection",
			expected: "https_connection",
		},
		{
			name:     "API abbreviation",
			input:    "TestAPI",
			expected: "api",
		},
		{
			name:     "RestAPIClient mixed",
			input:    "TestRestAPIClient",
			expected: "rest_api_client",
		},
		{
			name:     "DB abbreviation",
			input:    "TestDB",
			expected: "db",
		},
		{
			name:     "DBConnection mixed",
			input:    "TestDBConnection",
			expected: "db_connection",
		},
		{
			name:     "SQL abbreviation",
			input:    "TestSQL",
			expected: "sql",
		},
		{
			name:     "SQLQuery mixed",
			input:    "TestSQLQuery",
			expected: "sql_query",
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
		{
			name:     "AWS abbreviation",
			input:    "TestAWS",
			expected: "aws",
		},
		{
			name:     "AWSLambda mixed",
			input:    "TestAWSLambda",
			expected: "aws_lambda",
		},
		{
			name:     "IO abbreviation",
			input:    "TestIO",
			expected: "io",
		},
		{
			name:     "FileIO mixed",
			input:    "TestFileIO",
			expected: "file_io",
		},
		{
			name:     "EOF abbreviation",
			input:    "TestEOF",
			expected: "eof",
		},
		{
			name:     "CheckEOF mixed",
			input:    "TestCheckEOF",
			expected: "check_eof",
		},
		{
			name:     "UI abbreviation",
			input:    "TestUI",
			expected: "ui",
		},
		{
			name:     "UIComponent mixed",
			input:    "TestUIComponent",
			expected: "ui_component",
		},
		{
			name:     "multiple abbreviations in sequence",
			input:    "TestJSONAPIURL",
			expected: "json_api_url",
		},
		{
			name:     "TCP abbreviation",
			input:    "TestTCP",
			expected: "tcp",
		},
		{
			name:     "TCPIPConnection",
			input:    "TestTCPIPConnection",
			expected: "tcp_ip_connection",
		},
		{
			name:     "JWT abbreviation",
			input:    "TestJWT",
			expected: "jwt",
		},
		{
			name:     "JWTToken mixed",
			input:    "TestJWTToken",
			expected: "jwt_token",
		},
		{
			name:     "CLI abbreviation",
			input:    "TestCLI",
			expected: "cli",
		},
		{
			name:     "CLICommand mixed",
			input:    "TestCLICommand",
			expected: "cli_command",
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
