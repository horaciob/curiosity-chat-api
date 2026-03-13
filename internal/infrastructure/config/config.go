package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration.
type Config struct {
	ServerPort     string
	DSN            string
	InternalAPIKey string
	UserAPIURL     string
	LogLevel       string
	LogFormat      string
	AllowedOrigins []string
}

// Load reads environment variables and returns a Config.
func Load() *Config {
	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "8081"),
		DSN:            buildDSN(),
		InternalAPIKey: mustEnv("INTERNAL_API_KEY"),
		UserAPIURL:     getEnv("USER_API_URL", "http://localhost:8084"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		LogFormat:      getEnv("LOG_FORMAT", "text"),
		AllowedOrigins: getAllowedOrigins(),
	}
}

func getAllowedOrigins() []string {
	origins := getEnv("ALLOWED_ORIGINS", "")
	if origins == "" {
		// Default to localhost for development
		return []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"https://localhost:3000",
			"https://localhost:8080",
		}
	}
	// Split comma-separated origins
	var result []string
	for _, origin := range splitAndTrim(origins, ",") {
		if origin != "" {
			result = append(result, origin)
		}
	}
	return result
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitString(s, sep) {
		trimmed := ""
		for i := 0; i < len(part); i++ {
			if part[i] != ' ' && part[i] != '\t' {
				trimmed += string(part[i])
			}
		}
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i < len(s)-len(sep)+1 && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func buildDSN() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5434")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "curiosity_chat")
	sslmode := getEnv("DB_SSL_MODE", "disable")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return v
}
