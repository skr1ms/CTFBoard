package jwt

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	revokedKeyPrefix     = "jwt:revoked:"
	userRevokedKeyPrefix = "jwt:user_revoked_at:"
)

// ErrEmptyJTI is returned when Revoke is called with an empty jti.
var ErrEmptyJTI = errors.New("jwt: jti is required for revocation")

type RevocationStore interface {
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
	// RevokeUserTokens marks all tokens issued to userID before now as revoked.
	// ttl is how long the entry lives; use the maximum token TTL.
	RevokeUserTokens(ctx context.Context, userID uuid.UUID, ttl time.Duration) error
	// IsUserRevoked returns true when the token's IssuedAt is at or before the
	// user-level revocation timestamp stored by RevokeUserTokens.
	IsUserRevoked(ctx context.Context, userID uuid.UUID, issuedAt int64) (bool, error)
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

// RevokeUserTokens stores the current Unix timestamp for the given user so
// any token issued at or before that moment is considered revoked.
func (s *RedisRevocationStore) RevokeUserTokens(ctx context.Context, userID uuid.UUID, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return nil
	}
	if ttl < time.Second {
		ttl = time.Hour * 24 * 7
	}
	key := userRevokedKeyPrefix + userID.String()
	return s.client.Set(ctx, key, strconv.FormatInt(time.Now().Unix(), 10), ttl).Err()
}

// IsUserRevoked returns true when issuedAt is at or before the stored
// user-level revocation timestamp.
func (s *RedisRevocationStore) IsUserRevoked(ctx context.Context, userID uuid.UUID, issuedAt int64) (bool, error) {
	if s == nil || s.client == nil {
		return false, nil
	}
	key := userRevokedKeyPrefix + userID.String()
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("jwt user revocation check: %w", err)
	}
	revokedAt, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return false, fmt.Errorf("jwt user revocation parse: %w", err)
	}
	return issuedAt <= revokedAt, nil
}
