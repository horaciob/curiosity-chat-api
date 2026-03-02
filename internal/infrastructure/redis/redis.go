package redis

import (
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// NewClient creates a new Redis client from the given configuration.
func NewClient(cfg RedisConfig) *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}
