#!/bin/bash

# Clean up any existing generated files
rm -f *.go

# Create sample.go
cat > sample.go << 'EOF'
package example

import (
	"fmt"
	"strings"
)

// User represents a user in the system
type User struct {
	ID    int
	Name  string
	Email string
}

// GetUserByID retrieves a user by their ID
func GetUserByID(id int) (*User, error) {
	// Mock implementation
	if id <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", id)
	}
	return &User{
		ID:    id,
		Name:  fmt.Sprintf("User%d", id),
		Email: fmt.Sprintf("user%d@example.com", id),
	}, nil
}

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
	// Simple validation
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// FormatUserName formats a user's name for display
func FormatUserName(firstName, lastName string) string {
	return fmt.Sprintf("%s %s", strings.Title(firstName), strings.Title(lastName))
}

// private helper function
func sanitizeInput(input string) string {
	// Remove leading/trailing spaces
	return strings.TrimSpace(input)
}

// CalculateAge calculates age from birth year
func CalculateAge(birthYear, currentYear int) int {
	return currentYear - birthYear
}
EOF

# Create sample_test.go
cat > sample_test.go << 'EOF'
package example

import (
	"testing"
)

// TestGetUserByID tests the GetUserByID function
func TestGetUserByID(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		wantErr bool
	}{
		{"Valid ID", 1, false},
		{"Invalid ID", 0, true},
		{"Negative ID", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := GetUserByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && user == nil {
				t.Error("GetUserByID() returned nil user for valid ID")
			}
		})
	}
}

// TestValidateEmail tests email validation
func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"test.user@domain.co.jp", true},
		{"invalid", false},
		{"no-at-sign.com", false},
		{"no-dot@example", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			if got := ValidateEmail(tt.email); got != tt.want {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

// TestFormatUserName tests name formatting
func TestFormatUserName(t *testing.T) {
	tests := []struct {
		firstName string
		lastName  string
		want      string
	}{
		{"john", "doe", "John Doe"},
		{"JANE", "SMITH", "JANE SMITH"},
		{"", "", " "},
	}

	for _, tt := range tests {
		t.Run(tt.firstName+"_"+tt.lastName, func(t *testing.T) {
			if got := FormatUserName(tt.firstName, tt.lastName); got != tt.want {
				t.Errorf("FormatUserName(%q, %q) = %q, want %q", tt.firstName, tt.lastName, got, tt.want)
			}
		})
	}
}

// TestCalculateAge tests age calculation
func TestCalculateAge(t *testing.T) {
	tests := []struct {
		birthYear   int
		currentYear int
		want        int
	}{
		{1990, 2024, 34},
		{2000, 2024, 24},
		{2024, 2024, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := CalculateAge(tt.birthYear, tt.currentYear); got != tt.want {
				t.Errorf("CalculateAge(%d, %d) = %d, want %d", tt.birthYear, tt.currentYear, got, tt.want)
			}
		})
	}
}

// BenchmarkValidateEmail benchmarks email validation
func BenchmarkValidateEmail(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateEmail("user@example.com")
	}
}

// helper function for tests (not a test itself)
func setupTestData() *User {
	return &User{
		ID:    1,
		Name:  "Test User",
		Email: "test@example.com",
	}
}
EOF

echo "✅ サンプルファイルを作成しました:"
echo "  - sample.go"
echo "  - sample_test.go"