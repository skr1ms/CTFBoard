package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const revokedKeyPrefix = "jwt:revoked:"

// ErrEmptyJTI is returned when Revoke is called with an empty jti.
var ErrEmptyJTI = errors.New("jwt: jti is required for revocation")

type RevocationStore interface {
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

type RedisRevocationStore struct {
	client *redis.Client
}

func NewRedisRevocationStore(client *redis.Client) *RedisRevocationStore {
	return &RedisRevocationStore{client: client}
}

func (s *RedisRevocationStore) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" {
		return ErrEmptyJTI
	}
	if s == nil || s.client == nil {
		return nil
	}
	key := revokedKeyPrefix + jti
	if ttl < time.Second {
		ttl = time.Hour * 24 * 7
	}
	return s.client.Set(ctx, key, "1", ttl).Err()
}

func (s *RedisRevocationStore) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if s == nil || s.client == nil || jti == "" {
		return false, nil
	}
	key := revokedKeyPrefix + jti
	n, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("jwt revocation check: %w", err)
	}
	return n > 0, nil
}
