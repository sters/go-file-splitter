#!/bin/bash

# Clean up any existing generated files
rm -f *.go

# Create sample.go
cat > sample.go << 'EOF'
package example

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// Global constants
const (
	// Application configuration
	AppName    = "UserManagementSystem"
	AppVersion = "1.0.0"
	MaxRetries = 3

	// User roles
	RoleAdmin     = "admin"
	RoleModerator = "moderator"
	RoleUser      = "user"
	RoleGuest     = "guest"

	// Status codes
	StatusActive    = "active"
	StatusInactive  = "inactive"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"

	// Limits
	MaxLoginAttempts = 5
	SessionTimeout   = 30 * time.Minute
	MaxUploadSize    = 10 * 1024 * 1024 // 10MB
)

// Global variables
var (
	// Database instance
	DB *Database

	// Logger instance
	Logger = log.New(log.Writer(), "[APP] ", log.LstdFlags)

	// Metrics
	TotalUsers   int64
	ActiveUsers  int64
	metricsMutex sync.RWMutex

	// Error definitions
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrSessionExpired   = errors.New("session expired")
	ErrPermissionDenied = errors.New("permission denied")
	ErrDuplicateEmail   = errors.New("email already exists")
)

// Authenticator interface defines authentication behavior
type Authenticator interface {
	Authenticate(username, password string) (*User, error)
	ValidateToken(token string) (*Session, error)
	RefreshToken(token string) (string, error)
	Logout(token string) error
}

// Authorizer interface defines authorization behavior
type Authorizer interface {
	Authorize(user *User, resource string, action string) bool
	HasRole(user *User, role string) bool
	GetPermissions(user *User) []Permission
}

// Notifier interface for sending notifications
type Notifier interface {
	SendEmail(to string, subject string, body string) error
	SendSMS(to string, message string) error
	SendPush(userID int, title string, message string) error
}

// Repository interface for data access
type Repository interface {
	Create(entity interface{}) error
	Update(entity interface{}) error
	Delete(id int) error
	FindByID(id int) (interface{}, error)
	FindAll() ([]interface{}, error)
}

// User represents a user account with full details
type User struct {
	ID             int       `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Role           string    `json:"role"`
	Status         string    `json:"status"`
	LoginAttempts  int       `json:"-"`
	LastLoginAt    *time.Time `json:"last_login_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	EmailVerified  bool      `json:"email_verified"`
	TwoFactorEnabled bool    `json:"two_factor_enabled"`
	Preferences    UserPreferences `json:"preferences"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// UserPreferences stores user preferences
type UserPreferences struct {
	Language       string `json:"language"`
	Timezone       string `json:"timezone"`
	Theme          string `json:"theme"`
	Notifications  NotificationSettings `json:"notifications"`
}

// NotificationSettings defines notification preferences
type NotificationSettings struct {
	Email          bool `json:"email"`
	SMS            bool `json:"sms"`
	Push           bool `json:"push"`
	Marketing      bool `json:"marketing"`
}

// Session represents a user session
type Session struct {
	ID           string    `json:"id"`
	UserID       int       `json:"user_id"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
}

// Permission represents a system permission
type Permission struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

// Group represents a user group
type Group struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Members     []int     `json:"members"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   int       `json:"created_by"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        int64                  `json:"id"`
	UserID    int                    `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	IPAddress string                 `json:"ip_address"`
	Details   map[string]interface{} `json:"details"`
	Timestamp time.Time              `json:"timestamp"`
}

// Database represents a database connection
type Database struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	pool     *sync.Pool
	mutex    sync.RWMutex
}

// Token represents an authentication token
type Token struct {
	Value     string
	Type      string
	ExpiresAt time.Time
}

// PasswordPolicy defines password requirements
type PasswordPolicy struct {
	MinLength          int
	RequireUppercase   bool
	RequireLowercase   bool
	RequireNumbers     bool
	RequireSpecialChar bool
}

// RateLimiter tracks rate limiting
type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

// Cache interface for caching
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, expiration time.Duration)
	Delete(key string)
	Clear()
}

// User methods
// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
}

// GetDisplayName returns the display name for UI
func (u *User) GetDisplayName() string {
	if u.FirstName != "" || u.LastName != "" {
		return u.GetFullName()
	}
	if u.Username != "" {
		return u.Username
	}
	return u.Email
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsModerator checks if the user has moderator role
func (u *User) IsModerator() bool {
	return u.Role == RoleModerator
}

// IsActive checks if the user account is active
func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

// CanLogin checks if the user can attempt login
func (u *User) CanLogin() bool {
	return u.IsActive() && u.LoginAttempts < MaxLoginAttempts
}

// UpdateLastLogin updates the last login timestamp
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.LoginAttempts = 0
	u.UpdatedAt = now
}

// IncrementLoginAttempts increments failed login attempts
func (u *User) IncrementLoginAttempts() {
	u.LoginAttempts++
	if u.LoginAttempts >= MaxLoginAttempts {
		u.Status = StatusSuspended
	}
	u.UpdatedAt = time.Now()
}

// ResetLoginAttempts resets login attempts
func (u *User) ResetLoginAttempts() {
	u.LoginAttempts = 0
	if u.Status == StatusSuspended && u.LoginAttempts == 0 {
		u.Status = StatusActive
	}
	u.UpdatedAt = time.Now()
}

// HasPermission checks if user has a specific permission
func (u *User) HasPermission(permission string) bool {
	switch u.Role {
	case RoleAdmin:
		return true // Admins have all permissions
	case RoleModerator:
		// Moderators have limited permissions
		return permission != "system:manage" && permission != "user:delete"
	case RoleUser:
		// Regular users have basic permissions
		return strings.HasPrefix(permission, "user:read") || strings.HasPrefix(permission, "user:update:self")
	default:
		return false
	}
}

// UpdatePassword updates the user's password hash
func (u *User) UpdatePassword(newPasswordHash string) {
	u.PasswordHash = newPasswordHash
	u.UpdatedAt = time.Now()
}

// SetMetadata sets a metadata key-value pair
func (u *User) SetMetadata(key string, value interface{}) {
	if u.Metadata == nil {
		u.Metadata = make(map[string]interface{})
	}
	u.Metadata[key] = value
	u.UpdatedAt = time.Now()
}

// GetMetadata retrieves a metadata value by key
func (u *User) GetMetadata(key string) (interface{}, bool) {
	if u.Metadata == nil {
		return nil, false
	}
	value, exists := u.Metadata[key]
	return value, exists
}

// Session methods
// IsValid checks if the session is still valid
func (s *Session) IsValid() bool {
	return time.Now().Before(s.ExpiresAt)
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Extend extends the session expiration
func (s *Session) Extend(duration time.Duration) {
	s.ExpiresAt = time.Now().Add(duration)
	s.LastActivity = time.Now()
}

// UpdateActivity updates the last activity timestamp
func (s *Session) UpdateActivity() {
	s.LastActivity = time.Now()
}

// GetRemainingTime returns the remaining time before expiration
func (s *Session) GetRemainingTime() time.Duration {
	return time.Until(s.ExpiresAt)
}

// Database methods
// Connect establishes a database connection
func (db *Database) Connect() error {
	// Implementation would establish actual database connection
	Logger.Printf("Connecting to database %s:%d/%s", db.Host, db.Port, db.Name)
	return nil
}

// Close closes the database connection
func (db *Database) Close() error {
	Logger.Println("Closing database connection")
	return nil
}

// Execute executes a database query
func (db *Database) Execute(query string, args ...interface{}) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	// Implementation would execute actual query
	return nil
}

// Group methods
// AddMember adds a user to the group
func (g *Group) AddMember(userID int) {
	for _, id := range g.Members {
		if id == userID {
			return // Already a member
		}
	}
	g.Members = append(g.Members, userID)
}

// RemoveMember removes a user from the group
func (g *Group) RemoveMember(userID int) {
	for i, id := range g.Members {
		if id == userID {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)
			return
		}
	}
}

// HasMember checks if a user is a member
func (g *Group) HasMember(userID int) bool {
	for _, id := range g.Members {
		if id == userID {
			return true
		}
	}
	return false
}

// RateLimiter methods
// Allow checks if a request is allowed
func (r *RateLimiter) Allow(key string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	// Clean old requests
	requests := r.requests[key]
	validRequests := []time.Time{}
	for _, t := range requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) < r.limit {
		validRequests = append(validRequests, now)
		r.requests[key] = validRequests
		return true
	}

	r.requests[key] = validRequests
	return false
}

// Reset resets the rate limiter for a key
func (r *RateLimiter) Reset(key string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.requests, key)
}

// Global functions
// InitializeApp initializes the application
func InitializeApp(config map[string]string) error {
	Logger.Println("Initializing application...")

	// Initialize database
	DB = &Database{
		Host:     config["db_host"],
		Port:     5432,
		Name:     config["db_name"],
		User:     config["db_user"],
		Password: config["db_password"],
	}

	if err := DB.Connect(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	Logger.Printf("Application %s v%s initialized successfully", AppName, AppVersion)
	return nil
}

// GetUserByID retrieves a user by their ID
func GetUserByID(id int) (*User, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", id)
	}

	// Mock implementation
	user := &User{
		ID:        id,
		Username:  fmt.Sprintf("user%d", id),
		Email:     fmt.Sprintf("user%d@example.com", id),
		FirstName: fmt.Sprintf("First%d", id),
		LastName:  fmt.Sprintf("Last%d", id),
		Role:      RoleUser,
		Status:    StatusActive,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func GetUserByEmail(email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !ValidateEmail(email) {
		return nil, errors.New("invalid email format")
	}

	// Mock implementation
	return &User{
		ID:        1,
		Username:  "johndoe",
		Email:     email,
		FirstName: "John",
		LastName:  "Doe",
		Role:      RoleUser,
		Status:    StatusActive,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}, nil
}

// CreateUser creates a new user
func CreateUser(username, email, password string) (*User, error) {
	// Validate input
	if username == "" || email == "" || password == "" {
		return nil, errors.New("missing required fields")
	}

	if !ValidateEmail(email) {
		return nil, errors.New("invalid email format")
	}

	if !ValidatePassword(password) {
		return nil, errors.New("password does not meet requirements")
	}

	// Hash password
	passwordHash := HashPassword(password)

	// Create user
	user := &User{
		ID:           generateID(),
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         RoleUser,
		Status:       StatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Update metrics
	UpdateUserMetrics(1)

	return user, nil
}

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}

	// Simple validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	if parts[0] == "" || parts[1] == "" {
		return false
	}

	// Check domain part has at least one dot
	return strings.Contains(parts[1], ".")
}

// ValidatePassword validates password strength
func ValidatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
}

// HashPassword hashes a password
func HashPassword(password string) string {
	hasher := md5.New()
	hasher.Write([]byte(password))
	return hex.EncodeToString(hasher.Sum(nil))
}

// VerifyPassword verifies a password against its hash
func VerifyPassword(password, hash string) bool {
	return HashPassword(password) == hash
}

// GenerateSessionToken generates a new session token
func GenerateSessionToken() string {
	timestamp := time.Now().Unix()
	data := fmt.Sprintf("%d-%d", timestamp, generateID())
	hasher := md5.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}

// CreateSession creates a new user session
func CreateSession(userID int, ipAddress, userAgent string) (*Session, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user ID")
	}

	session := &Session{
		ID:           fmt.Sprintf("sess_%d", generateID()),
		UserID:       userID,
		Token:        GenerateSessionToken(),
		RefreshToken: GenerateSessionToken(),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(SessionTimeout),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	return session, nil
}

// LogAction logs an action to the audit log
func LogAction(userID int, action, resource string, details map[string]interface{}) {
	log := &AuditLog{
		ID:        generateID(),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		Timestamp: time.Now(),
	}

	Logger.Printf("Audit: User %d performed %s on %s", userID, action, resource)
	_ = log // Would normally save to database
}

// UpdateUserMetrics updates global user metrics
func UpdateUserMetrics(delta int64) {
	metricsMutex.Lock()
	defer metricsMutex.Unlock()

	TotalUsers += delta
	if delta > 0 {
		ActiveUsers += delta
	}
}

// GetUserMetrics returns current user metrics
func GetUserMetrics() (total, active int64) {
	metricsMutex.RLock()
	defer metricsMutex.RUnlock()

	return TotalUsers, ActiveUsers
}

// FormatUserName formats a user's name for display
func FormatUserName(firstName, lastName string) string {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)

	if firstName == "" && lastName == "" {
		return "Anonymous"
	}

	return fmt.Sprintf("%s %s", strings.Title(firstName), strings.Title(lastName))
}

// CalculateAge calculates age from birth year
func CalculateAge(birthYear, currentYear int) int {
	if birthYear <= 0 || currentYear < birthYear {
		return 0
	}
	return currentYear - birthYear
}

// SendWelcomeEmail sends a welcome email to a new user
func SendWelcomeEmail(user *User) error {
	if !ValidateEmail(user.Email) {
		return errors.New("invalid email address")
	}

	subject := fmt.Sprintf("Welcome to %s, %s!", AppName, user.GetDisplayName())
	body := fmt.Sprintf("Thank you for joining %s. Your account has been created successfully.", AppName)

	Logger.Printf("Sending welcome email to %s", user.Email)
	// Would normally send actual email
	_ = subject
	_ = body

	return nil
}

// Private helper functions
// generateID generates a unique ID
func generateID() int64 {
	return time.Now().UnixNano()
}

// sanitizeInput sanitizes user input
func sanitizeInput(input string) string {
	// Remove leading/trailing spaces and normalize
	input = strings.TrimSpace(input)
	// Remove multiple spaces
	input = strings.Join(strings.Fields(input), " ")
	// Remove potentially dangerous characters
	input = strings.ReplaceAll(input, "<", "")
	input = strings.ReplaceAll(input, ">", "")
	return input
}

// isValidUsername checks if username is valid
func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 20 {
		return false
	}

	for _, char := range username {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-') {
			return false
		}
	}

	return true
}

// normalizeEmail normalizes an email address
func normalizeEmail(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	// Remove dots from Gmail addresses
	if strings.HasSuffix(parts[1], "gmail.com") {
		parts[0] = strings.ReplaceAll(parts[0], ".", "")
		// Remove everything after + in email
		if idx := strings.Index(parts[0], "+"); idx != -1 {
			parts[0] = parts[0][:idx]
		}
	}

	return parts[0] + "@" + parts[1]
}
EOF

# Create sample_test.go
cat > sample_test.go << 'EOF'
package example

import (
	"testing"
	"time"
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

// TestGetUserByEmail tests the GetUserByEmail function
func TestGetUserByEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"Valid email", "user@example.com", false},
		{"Another valid email", "test@domain.co.jp", false},
		{"Invalid email format", "not-an-email", true},
		{"Empty email", "", true},
		{"Missing domain", "user@", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := GetUserByEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && user == nil {
				t.Error("GetUserByEmail() returned nil user for valid email")
			}
		})
	}
}

// TestCreateUser tests user creation
func TestCreateUser(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		password string
		wantErr  bool
	}{
		{"Valid user", "john_doe", "john@example.com", "SecurePass123!", false},
		{"Missing username", "", "john@example.com", "SecurePass123!", true},
		{"Invalid email", "john_doe", "invalid-email", "SecurePass123!", true},
		{"Weak password", "john_doe", "john@example.com", "weak", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := CreateUser(tt.username, tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && user == nil {
				t.Error("CreateUser() returned nil user for valid input")
			}
			if !tt.wantErr && user != nil {
				if user.Username != tt.username {
					t.Errorf("CreateUser() username = %v, want %v", user.Username, tt.username)
				}
				if user.Email != tt.email {
					t.Errorf("CreateUser() email = %v, want %v", user.Email, tt.email)
				}
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
		{"user+tag@example.com", true},
		{"invalid", false},
		{"no-at-sign.com", false},
		{"no-dot@example", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
		{"  user@example.com  ", true}, // Should handle trimming
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			if got := ValidateEmail(tt.email); got != tt.want {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

// TestValidatePassword tests password validation
func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		want     bool
	}{
		{"ValidPass123!", true},
		{"ComplexP@ssw0rd", true},
		{"short", false},
		{"nouppercase123!", false},
		{"NOLOWERCASE123!", false},
		{"NoNumbers!", false},
		{"NoSpecialChar123", false},
		{"12345678", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			if got := ValidatePassword(tt.password); got != tt.want {
				t.Errorf("ValidatePassword(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}

// TestHashPassword tests password hashing
func TestHashPassword(t *testing.T) {
	password := "TestPassword123"
	hash1 := HashPassword(password)
	hash2 := HashPassword(password)

	if hash1 != hash2 {
		t.Error("HashPassword should produce consistent results for the same input")
	}

	if hash1 == password {
		t.Error("HashPassword should not return the original password")
	}

	if len(hash1) != 32 { // MD5 produces 32 character hex string
		t.Errorf("HashPassword should return 32 character hash, got %d", len(hash1))
	}
}

// TestVerifyPassword tests password verification
func TestVerifyPassword(t *testing.T) {
	password := "TestPassword123"
	hash := HashPassword(password)

	if !VerifyPassword(password, hash) {
		t.Error("VerifyPassword should return true for correct password")
	}

	if VerifyPassword("WrongPassword", hash) {
		t.Error("VerifyPassword should return false for incorrect password")
	}
}

// TestCreateSession tests session creation
func TestCreateSession(t *testing.T) {
	tests := []struct {
		name      string
		userID    int
		ipAddress string
		userAgent string
		wantErr   bool
	}{
		{"Valid session", 1, "192.168.1.1", "Mozilla/5.0", false},
		{"Invalid user ID", 0, "192.168.1.1", "Mozilla/5.0", true},
		{"Negative user ID", -1, "192.168.1.1", "Mozilla/5.0", true},
		{"Empty IP", 10, "", "Mozilla/5.0", false},
		{"Empty user agent", 10, "192.168.1.1", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := CreateSession(tt.userID, tt.ipAddress, tt.userAgent)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && session == nil {
				t.Error("CreateSession() returned nil session for valid input")
			}
			if !tt.wantErr && session != nil {
				if session.UserID != tt.userID {
					t.Errorf("CreateSession() userID = %v, want %v", session.UserID, tt.userID)
				}
				if session.Token == "" {
					t.Error("CreateSession() generated empty token")
				}
				if session.RefreshToken == "" {
					t.Error("CreateSession() generated empty refresh token")
				}
			}
		})
	}
}

// TestUserMethods tests User struct methods
func TestUserMethods(t *testing.T) {
	user := &User{
		ID:        1,
		Username:  "johndoe",
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      RoleAdmin,
		Status:    StatusActive,
		LoginAttempts: 2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	t.Run("GetFullName", func(t *testing.T) {
		want := "John Doe"
		if got := user.GetFullName(); got != want {
			t.Errorf("GetFullName() = %v, want %v", got, want)
		}
	})

	t.Run("GetDisplayName", func(t *testing.T) {
		want := "John Doe"
		if got := user.GetDisplayName(); got != want {
			t.Errorf("GetDisplayName() = %v, want %v", got, want)
		}
	})

	t.Run("IsAdmin", func(t *testing.T) {
		if !user.IsAdmin() {
			t.Error("IsAdmin() should return true for admin role")
		}
	})

	t.Run("IsActive", func(t *testing.T) {
		if !user.IsActive() {
			t.Error("IsActive() should return true for active status")
		}
	})

	t.Run("CanLogin", func(t *testing.T) {
		if !user.CanLogin() {
			t.Error("CanLogin() should return true for active user with attempts < max")
		}
	})

	t.Run("UpdateLastLogin", func(t *testing.T) {
		user.UpdateLastLogin()
		if user.LastLoginAt == nil {
			t.Error("UpdateLastLogin() should set LastLoginAt")
		}
		if user.LoginAttempts != 0 {
			t.Error("UpdateLastLogin() should reset LoginAttempts")
		}
	})

	t.Run("IncrementLoginAttempts", func(t *testing.T) {
		initialAttempts := user.LoginAttempts
		user.IncrementLoginAttempts()
		if user.LoginAttempts != initialAttempts+1 {
			t.Error("IncrementLoginAttempts() should increment attempts")
		}
	})

	t.Run("HasPermission", func(t *testing.T) {
		if !user.HasPermission("system:manage") {
			t.Error("Admin should have all permissions")
		}
	})

	t.Run("SetMetadata and GetMetadata", func(t *testing.T) {
		key := "test_key"
		value := "test_value"
		user.SetMetadata(key, value)

		got, exists := user.GetMetadata(key)
		if !exists {
			t.Error("GetMetadata() should find the set key")
		}
		if got != value {
			t.Errorf("GetMetadata() = %v, want %v", got, value)
		}
	})
}

// TestSessionMethods tests Session struct methods
func TestSessionMethods(t *testing.T) {
	session := &Session{
		ID:           "sess_123",
		UserID:       1,
		Token:        "token_123",
		RefreshToken: "refresh_123",
		ExpiresAt:    time.Now().Add(30 * time.Minute),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	t.Run("IsValid", func(t *testing.T) {
		if !session.IsValid() {
			t.Error("IsValid() should return true for non-expired session")
		}
	})

	t.Run("IsExpired", func(t *testing.T) {
		if session.IsExpired() {
			t.Error("IsExpired() should return false for non-expired session")
		}
	})

	t.Run("Extend", func(t *testing.T) {
		originalExpiry := session.ExpiresAt
		session.Extend(1 * time.Hour)
		if !session.ExpiresAt.After(originalExpiry) {
			t.Error("Extend() should extend expiration time")
		}
	})

	t.Run("GetRemainingTime", func(t *testing.T) {
		remaining := session.GetRemainingTime()
		if remaining <= 0 {
			t.Error("GetRemainingTime() should return positive duration for valid session")
		}
	})

	t.Run("UpdateActivity", func(t *testing.T) {
		originalActivity := session.LastActivity
		time.Sleep(10 * time.Millisecond)
		session.UpdateActivity()
		if !session.LastActivity.After(originalActivity) {
			t.Error("UpdateActivity() should update LastActivity timestamp")
		}
	})
}

// TestGroupMethods tests Group struct methods
func TestGroupMethods(t *testing.T) {
	group := &Group{
		ID:          1,
		Name:        "Admins",
		Description: "Administrator group",
		Members:     []int{1, 2, 3},
		Permissions: []string{"admin:all"},
		CreatedAt:   time.Now(),
	}

	t.Run("AddMember", func(t *testing.T) {
		group.AddMember(4)
		if !group.HasMember(4) {
			t.Error("AddMember() should add new member")
		}

		// Test duplicate addition
		initialLen := len(group.Members)
		group.AddMember(4)
		if len(group.Members) != initialLen {
			t.Error("AddMember() should not add duplicate members")
		}
	})

	t.Run("RemoveMember", func(t *testing.T) {
		group.RemoveMember(2)
		if group.HasMember(2) {
			t.Error("RemoveMember() should remove member")
		}
	})

	t.Run("HasMember", func(t *testing.T) {
		if !group.HasMember(1) {
			t.Error("HasMember() should return true for existing member")
		}
		if group.HasMember(99) {
			t.Error("HasMember() should return false for non-existing member")
		}
	})
}

// TestRateLimiterMethods tests RateLimiter struct methods
func TestRateLimiterMethods(t *testing.T) {
	limiter := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    3,
		window:   1 * time.Second,
	}

	t.Run("Allow", func(t *testing.T) {
		key := "test_key"

		// Should allow first few requests
		for i := 0; i < 3; i++ {
			if !limiter.Allow(key) {
				t.Errorf("Allow() should return true for request %d", i+1)
			}
		}

		// Should block after limit
		if limiter.Allow(key) {
			t.Error("Allow() should return false after limit reached")
		}
	})

	t.Run("Reset", func(t *testing.T) {
		key := "reset_key"

		// Fill up the limit
		for i := 0; i < 3; i++ {
			limiter.Allow(key)
		}

		// Reset and try again
		limiter.Reset(key)
		if !limiter.Allow(key) {
			t.Error("Allow() should return true after reset")
		}
	})
}

// TestDatabaseMethods tests Database struct methods
func TestDatabaseMethods(t *testing.T) {
	db := &Database{
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		User:     "testuser",
		Password: "testpass",
	}

	t.Run("Connect", func(t *testing.T) {
		err := db.Connect()
		if err != nil {
			t.Errorf("Connect() error = %v", err)
		}
	})

	t.Run("Execute", func(t *testing.T) {
		err := db.Execute("SELECT * FROM users WHERE id = ?", 1)
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := db.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
}

// TestInitializeApp tests application initialization
func TestInitializeApp(t *testing.T) {
	config := map[string]string{
		"db_host":     "localhost",
		"db_name":     "testdb",
		"db_user":     "testuser",
		"db_password": "testpass",
	}

	err := InitializeApp(config)
	if err != nil {
		t.Errorf("InitializeApp() error = %v", err)
	}

	if DB == nil {
		t.Error("InitializeApp() should initialize DB")
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
		{"", "", "Anonymous"},
		{"  john  ", "  doe  ", "John Doe"},
		{"mary", "", "Mary "},
		{"", "johnson", " Johnson"},
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
		{0, 2024, 0},      // Invalid birth year
		{2025, 2024, 0},   // Future birth year
		{-1, 2024, 0},     // Negative birth year
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := CalculateAge(tt.birthYear, tt.currentYear); got != tt.want {
				t.Errorf("CalculateAge(%d, %d) = %d, want %d", tt.birthYear, tt.currentYear, got, tt.want)
			}
		})
	}
}

// TestSendWelcomeEmail tests welcome email sending
func TestSendWelcomeEmail(t *testing.T) {
	tests := []struct {
		name    string
		user    *User
		wantErr bool
	}{
		{
			"Valid user",
			&User{
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
			false,
		},
		{
			"Invalid email",
			&User{
				Email:     "invalid-email",
				FirstName: "John",
				LastName:  "Doe",
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SendWelcomeEmail(tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendWelcomeEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestUpdateUserMetrics tests metric updates
func TestUpdateUserMetrics(t *testing.T) {
	// Reset metrics
	metricsMutex.Lock()
	TotalUsers = 0
	ActiveUsers = 0
	metricsMutex.Unlock()

	UpdateUserMetrics(5)
	total, active := GetUserMetrics()

	if total != 5 {
		t.Errorf("UpdateUserMetrics() total = %d, want %d", total, 5)
	}
	if active != 5 {
		t.Errorf("UpdateUserMetrics() active = %d, want %d", active, 5)
	}

	UpdateUserMetrics(-2)
	total, active = GetUserMetrics()

	if total != 3 {
		t.Errorf("UpdateUserMetrics() total after decrease = %d, want %d", total, 3)
	}
}

// BenchmarkValidateEmail benchmarks email validation
func BenchmarkValidateEmail(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateEmail("user@example.com")
	}
}

// BenchmarkHashPassword benchmarks password hashing
func BenchmarkHashPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashPassword("TestPassword123!")
	}
}

// BenchmarkUserCanLogin benchmarks login check
func BenchmarkUserCanLogin(b *testing.B) {
	user := &User{
		Status:        StatusActive,
		LoginAttempts: 2,
	}

	for i := 0; i < b.N; i++ {
		user.CanLogin()
	}
}

// Helper functions for tests
func setupTestUser() *User {
	return &User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      RoleUser,
		Status:    StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func setupTestSession() *Session {
	return &Session{
		ID:           "test_session",
		UserID:       1,
		Token:        "test_token",
		RefreshToken: "test_refresh",
		ExpiresAt:    time.Now().Add(30 * time.Minute),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
}

func setupTestGroup() *Group {
	return &Group{
		ID:          1,
		Name:        "Test Group",
		Description: "A test group",
		Members:     []int{1, 2, 3},
		Permissions: []string{"read", "write"},
		CreatedAt:   time.Now(),
	}
}
EOF

echo "✅ サンプルファイルを作成しました:"
echo "  - sample.go (複雑な構成: interfaces, methods, constants, variables, multiple types)"
echo "  - sample_test.go (包括的なテストケース)"
echo ""
echo "含まれる要素:"
echo "  • 4つのinterface (Authenticator, Authorizer, Notifier, Repository, Cache)"
echo "  • 11のstruct型 (User, Session, Permission, Group, AuditLog, Database, Token, etc.)"
echo "  • Userに13個のメソッド"
echo "  • Sessionに5個のメソッド"
echo "  • Databaseに3個のメソッド"
echo "  • Groupに3個のメソッド"
echo "  • RateLimiterに2個のメソッド"
echo "  • グローバル定数 (const) 複数定義"
echo "  • グローバル変数 (var) 複数定義"
echo "  • 15個以上のグローバル関数"
echo "  • 4個のプライベート関数"
echo "  • 包括的なテストケース"