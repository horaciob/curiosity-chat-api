package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadWithDefaults(t *testing.T) {
	// Set required env var
	os.Setenv("INTERNAL_API_KEY", "test-key")
	defer os.Unsetenv("INTERNAL_API_KEY")

	// Clear other env vars to test defaults
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("DB_SSL_MODE")
	os.Unsetenv("USER_API_URL")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("LOG_FORMAT")
	os.Unsetenv("ALLOWED_ORIGINS")

	cfg := Load()

	assert.Equal(t, "8081", cfg.ServerPort)
	assert.Equal(t, "test-key", cfg.InternalAPIKey)
	assert.Equal(t, "http://localhost:8084", cfg.UserAPIURL)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "console", cfg.LogFormat)
	assert.NotEmpty(t, cfg.DSN)
	assert.Contains(t, cfg.DSN, "host=localhost")
	assert.Contains(t, cfg.DSN, "port=5434")
	assert.Contains(t, cfg.DSN, "dbname=curiosity_chat")
	assert.Len(t, cfg.AllowedOrigins, 4)
	assert.Contains(t, cfg.AllowedOrigins, "http://localhost:3000")
}

func TestLoadWithCustomValues(t *testing.T) {
	os.Setenv("INTERNAL_API_KEY", "custom-key")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "admin")
	os.Setenv("DB_PASSWORD", "secret")
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("DB_SSL_MODE", "require")
	os.Setenv("USER_API_URL", "http://user-api.example.com")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "json")
	os.Setenv("ALLOWED_ORIGINS", "https://app1.com, https://app2.com")

	defer func() {
		os.Unsetenv("INTERNAL_API_KEY")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSL_MODE")
		os.Unsetenv("USER_API_URL")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("ALLOWED_ORIGINS")
	}()

	cfg := Load()

	assert.Equal(t, "9090", cfg.ServerPort)
	assert.Equal(t, "custom-key", cfg.InternalAPIKey)
	assert.Equal(t, "http://user-api.example.com", cfg.UserAPIURL)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "json", cfg.LogFormat)
	assert.Contains(t, cfg.DSN, "host=db.example.com")
	assert.Contains(t, cfg.DSN, "port=5432")
	assert.Contains(t, cfg.DSN, "user=admin")
	assert.Contains(t, cfg.DSN, "password=secret")
	assert.Contains(t, cfg.DSN, "dbname=test_db")
	assert.Contains(t, cfg.DSN, "sslmode=require")
	assert.Len(t, cfg.AllowedOrigins, 2)
	assert.Contains(t, cfg.AllowedOrigins, "https://app1.com")
	assert.Contains(t, cfg.AllowedOrigins, "https://app2.com")
}

func TestLoadPanicsWithoutInternalAPIKey(t *testing.T) {
	os.Unsetenv("INTERNAL_API_KEY")

	assert.Panics(t, func() {
		Load()
	})
}

func TestGetAllowedOriginsWithEmptyString(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	origins := getAllowedOrigins()

	assert.Len(t, origins, 4)
	assert.Contains(t, origins, "http://localhost:3000")
	assert.Contains(t, origins, "https://localhost:8080")
}

func TestGetAllowedOriginsWithSpaces(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "  https://app1.com  ,  https://app2.com  ")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	origins := getAllowedOrigins()

	assert.Len(t, origins, 2)
	assert.Contains(t, origins, "https://app1.com")
	assert.Contains(t, origins, "https://app2.com")
}

func TestGetAllowedOriginsWithTabs(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "https://app1.com\t,\thttps://app2.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	origins := getAllowedOrigins()

	assert.Len(t, origins, 2)
	assert.Contains(t, origins, "https://app1.com")
	assert.Contains(t, origins, "https://app2.com")
}

func TestGetAllowedOriginsWithEmptyEntries(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "https://app1.com,,https://app2.com,,,")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	origins := getAllowedOrigins()

	assert.Len(t, origins, 2)
	assert.Contains(t, origins, "https://app1.com")
	assert.Contains(t, origins, "https://app2.com")
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "simple comma separated",
			input:    "a,b,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with spaces",
			input:    "  a  ,  b  ,  c  ",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with tabs",
			input:    "a\t,\tb,\tc",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "mixed whitespace",
			input:    "  a\t , \t b , c  ",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty entries removed",
			input:    "a,,b,,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all whitespace removed",
			input:    "  ,  ,  ",
			sep:      ",",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndTrim(tt.input, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "simple split",
			input:    "a,b,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty string",
			input:    "",
			sep:      ",",
			expected: []string{""},
		},
		{
			name:     "no separator",
			input:    "abc",
			sep:      ",",
			expected: []string{"abc"},
		},
		{
			name:     "multi-char separator",
			input:    "a::b::c",
			sep:      "::",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "consecutive separators",
			input:    "a,,b",
			sep:      ",",
			expected: []string{"a", "", "b"},
		},
		{
			name:     "separator at end",
			input:    "a,b,",
			sep:      ",",
			expected: []string{"a", "b", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitString(tt.input, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildDSN(t *testing.T) {
	os.Setenv("DB_HOST", "testhost")
	os.Setenv("DB_PORT", "1234")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_SSL_MODE", "verify-full")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSL_MODE")
	}()

	dsn := buildDSN()

	assert.Contains(t, dsn, "host=testhost")
	assert.Contains(t, dsn, "port=1234")
	assert.Contains(t, dsn, "user=testuser")
	assert.Contains(t, dsn, "password=testpass")
	assert.Contains(t, dsn, "dbname=testdb")
	assert.Contains(t, dsn, "sslmode=verify-full")
}

func TestBuildDSNWithDefaults(t *testing.T) {
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("DB_SSL_MODE")

	dsn := buildDSN()

	assert.Contains(t, dsn, "host=localhost")
	assert.Contains(t, dsn, "port=5434")
	assert.Contains(t, dsn, "user=postgres")
	assert.Contains(t, dsn, "password=postgres")
	assert.Contains(t, dsn, "dbname=curiosity_chat")
	assert.Contains(t, dsn, "sslmode=disable")
}

func TestGetEnvWithValue(t *testing.T) {
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	result := getEnv("TEST_KEY", "default")
	assert.Equal(t, "test_value", result)
}

func TestGetEnvWithFallback(t *testing.T) {
	os.Unsetenv("NONEXISTENT_KEY")

	result := getEnv("NONEXISTENT_KEY", "default_value")
	assert.Equal(t, "default_value", result)
}

func TestGetEnvWithEmptyValue(t *testing.T) {
	os.Setenv("EMPTY_KEY", "")
	defer os.Unsetenv("EMPTY_KEY")

	result := getEnv("EMPTY_KEY", "default")
	assert.Equal(t, "default", result)
}

func TestMustEnvWithValue(t *testing.T) {
	os.Setenv("REQUIRED_KEY", "required_value")
	defer os.Unsetenv("REQUIRED_KEY")

	result := mustEnv("REQUIRED_KEY")
	assert.Equal(t, "required_value", result)
}

func TestMustEnvPanicsWithoutValue(t *testing.T) {
	os.Unsetenv("PANIC_KEY")

	assert.Panics(t, func() {
		mustEnv("PANIC_KEY")
	})
}
