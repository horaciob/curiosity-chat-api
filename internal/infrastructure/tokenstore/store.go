package tokenstore

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	RefreshTokenTTL    = 30 * 24 * time.Hour
	keyPrefix          = "rt:"
	userSetPrefix      = "rt:user:"
)

// ErrTokenNotFound is returned when a refresh token hash has no matching entry.
var ErrTokenNotFound = errors.New("refresh token not found or expired")

// Store defines the contract for refresh token persistence.
type Store interface {
	Save(ctx context.Context, userID, tokenHash string, ttl time.Duration) error
	Get(ctx context.Context, tokenHash string) (userID string, err error)
	Delete(ctx context.Context, tokenHash string) error
	DeleteAllForUser(ctx context.Context, userID string) error
}

// RedisStore implements Store using Redis.
//
// Key layout:
//   rt:{hash}           → userID  (TTL = 30 days)
//   rt:user:{userID}    → Redis Set of active hashes (for bulk revocation)
type RedisStore struct {
	client *goredis.Client
}

// NewRedisStore creates a RedisStore backed by the provided client.
func NewRedisStore(client *goredis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func tokenKey(hash string) string    { return keyPrefix + hash }
func userSetKey(userID string) string { return userSetPrefix + userID }

// Save persists the hash → userID mapping and tracks it in the user's set.
func (s *RedisStore) Save(ctx context.Context, userID, tokenHash string, ttl time.Duration) error {
	pipe := s.client.Pipeline()
	pipe.Set(ctx, tokenKey(tokenHash), userID, ttl)
	pipe.SAdd(ctx, userSetKey(userID), tokenHash)
	pipe.Expire(ctx, userSetKey(userID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// Get returns the userID associated with the hash, or ErrTokenNotFound.
func (s *RedisStore) Get(ctx context.Context, tokenHash string) (string, error) {
	val, err := s.client.Get(ctx, tokenKey(tokenHash)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrTokenNotFound
	}
	return val, err
}

// Delete removes a single token hash (used on rotation and sign-out of one session).
func (s *RedisStore) Delete(ctx context.Context, tokenHash string) error {
	// We don't bother removing the hash from the user set — it will expire naturally
	// or be cleaned up by DeleteAllForUser.
	return s.client.Del(ctx, tokenKey(tokenHash)).Err()
}

// DeleteAllForUser revokes all refresh tokens for a user (sign-out all sessions).
func (s *RedisStore) DeleteAllForUser(ctx context.Context, userID string) error {
	hashes, err := s.client.SMembers(ctx, userSetKey(userID)).Result()
	if err != nil {
		return err
	}
	if len(hashes) == 0 {
		return nil
	}
	keys := make([]string, 0, len(hashes)+1)
	for _, h := range hashes {
		keys = append(keys, tokenKey(h))
	}
	keys = append(keys, userSetKey(userID))
	return s.client.Del(ctx, keys...).Err()
}
