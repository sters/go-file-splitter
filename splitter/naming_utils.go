package splitter

import (
	"strings"
	"unicode"
)

func functionNameToSnakeCase(name string) string {
	// Handle common abbreviations
	commonAbbreviations := getCommonAbbreviations()
	for _, abbr := range commonAbbreviations {
		if strings.ToUpper(name) == abbr {
			return strings.ToLower(name)
		}
	}

	result := make([]rune, 0, len(name)*2)
	runes := []rune(name)

	for i := 0; i < len(runes); i++ {
		// Check if current position starts with a known abbreviation
		if abbr, length := matchesAbbreviation(runes, i); abbr != "" {
			// Add underscore before abbreviation if needed
			if i > 0 && len(result) > 0 && result[len(result)-1] != '_' {
				result = append(result, '_')
			}
			// Add the abbreviation in lowercase
			for _, r := range strings.ToLower(abbr) {
				result = append(result, r)
			}
			i += length - 1

			continue
		}

		// Handle regular character
		r := runes[i]
		if shouldAddUnderscore(runes, i, result) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}

	resultStr := string(result)
	if resultStr == "" {
		return "func"
	}

	// Remove leading underscore if present
	return strings.TrimLeft(resultStr, "_")
}

func testNameToSnakeCase(name string) string {
	if !strings.HasPrefix(name, "Test") {
		return strings.ToLower(name)
	}

	name = strings.TrimPrefix(name, "Test")
	name = strings.TrimLeft(name, "_")

	if name == "" {
		return "test"
	}

	// Check if the entire name is a common abbreviation
	commonAbbreviations := getCommonAbbreviations()
	for _, abbr := range commonAbbreviations {
		if strings.ToUpper(name) == abbr {
			return strings.ToLower(name)
		}
	}

	result := make([]rune, 0, len(name)*2)
	runes := []rune(name)

	for i := 0; i < len(runes); i++ {
		// Check if current position starts with a known abbreviation
		if abbr, length := matchesAbbreviation(runes, i); abbr != "" {
			// Add underscore before abbreviation if needed
			if i > 0 && len(result) > 0 && result[len(result)-1] != '_' {
				result = append(result, '_')
			}
			// Add the abbreviation in lowercase
			for _, r := range strings.ToLower(abbr) {
				result = append(result, r)
			}
			i += length - 1

			continue
		}

		// Handle regular character
		r := runes[i]
		if shouldAddUnderscore(runes, i, result) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}

	resultStr := string(result)
	if resultStr == "" {
		return "test"
	}

	return resultStr
}

func getCommonAbbreviations() []string {
	return []string{
		"ID", "UUID", "URL", "URI", "API", "HTTP", "HTTPS", "JSON", "XML", "CSV",
		"SQL", "DB", "TCP", "UDP", "IP", "DNS", "SSH", "TLS", "SSL", "JWT",
		"AWS", "GCP", "CPU", "GPU", "RAM", "ROM", "IO", "EOF", "TTL", "CDN",
		"HTML", "CSS", "JS", "MD5", "SHA", "RSA", "AES", "UTF", "ASCII",
		"CRUD", "REST", "RPC", "GRPC", "MQTT", "AMQP", "SMTP", "IMAP", "POP",
		"SDK", "CLI", "GUI", "UI", "UX", "OS", "VM", "PDF", "PNG", "JPG", "GIF",
	}
}

func matchesAbbreviation(runes []rune, i int) (string, int) {
	commonAbbreviations := getCommonAbbreviations()
	for _, abbr := range commonAbbreviations {
		if i+len(abbr) > len(runes) {
			continue
		}

		substr := string(runes[i : i+len(abbr)])
		if strings.ToUpper(substr) != abbr {
			continue
		}

		// Check if it's a word boundary
		atWordBoundary := i+len(abbr) == len(runes) ||
			(i+len(abbr) < len(runes) && unicode.IsUpper(runes[i+len(abbr)]))

		if atWordBoundary {
			return abbr, len(abbr)
		}
	}

	return "", 0
}

func methodNameToSnakeCase(receiverType, methodName string) string {
	// Convert both receiver type and method name to snake case and combine
	receiverSnake := functionNameToSnakeCase(receiverType)
	methodSnake := functionNameToSnakeCase(methodName)

	return receiverSnake + "_" + methodSnake
}

func shouldAddUnderscore(runes []rune, i int, result []rune) bool {
	if i == 0 || !unicode.IsUpper(runes[i]) {
		return false
	}

	if len(result) == 0 || result[len(result)-1] == '_' {
		return false
	}

	// Uppercase followed by lowercase
	if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
		return true
	}

	// Lowercase followed by uppercase
	if i > 0 && unicode.IsLower(runes[i-1]) {
		return true
	}

	return false
}
