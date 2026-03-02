package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration.
type Config struct {
	ServerPort      string
	DSN             string
	InternalAPIKey  string
	CuriosityAPIURL string
	AuthAPIURL      string
	LogLevel        string
	LogFormat       string
}

// Load reads environment variables and returns a Config.
func Load() *Config {
	return &Config{
		ServerPort:      getEnv("SERVER_PORT", "8081"),
		DSN:             buildDSN(),
		InternalAPIKey:  mustEnv("INTERNAL_API_KEY"),
		CuriosityAPIURL: getEnv("CURIOSITY_API_URL", "http://localhost:8080"),
		AuthAPIURL:      getEnv("AUTH_API_URL", "http://localhost:8082"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		LogFormat:       getEnv("LOG_FORMAT", "console"),
	}
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
