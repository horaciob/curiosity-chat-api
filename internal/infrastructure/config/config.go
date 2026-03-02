package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	ServerPort      string
	DSN             string
	JWTSecret       string
	InternalAPIKey  string
	CuriosityAPIURL string
	Redis           RedisConfig
	LogLevel        string
	LogFormat       string
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// Load reads environment variables and returns a Config.
func Load() *Config {
	return &Config{
		ServerPort:      getEnv("SERVER_PORT", "8081"),
		DSN:             buildDSN(),
		JWTSecret:       mustEnv("JWT_SECRET"),
		InternalAPIKey:  mustEnv("INTERNAL_API_KEY"),
		CuriosityAPIURL: getEnv("CURIOSITY_API_URL", "http://localhost:8080"),
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "console"),
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

func getEnvAsInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
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
