package loginlockout

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
)

const (
	defaultMax = 5
	defaultTTL = 15 * time.Minute
)

// Tracker counts failed login attempts per email and reports when over limit.
type Tracker struct {
	client *redis.Client
	max    int
	ttl    time.Duration
}

// NewTracker returns a Tracker that allows maxAttempts failed attempts per email within ttl.
// If maxAttempts or ttl is zero, defaults (5, 15min) are used.
func NewTracker(client *redis.Client, maxAttempts int, ttl time.Duration) *Tracker {
	if maxAttempts <= 0 {
		maxAttempts = defaultMax
	}

	if ttl <= 0 {
		ttl = defaultTTL
	}

	return &Tracker{client: client, max: maxAttempts, ttl: ttl}
}

// IsLocked returns true if the email has exceeded the failed attempt limit.
func (t *Tracker) IsLocked(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, nil
	}

	key := cache.KeyFailedLoginPrefix + email

	n, err := t.client.Get(ctx, key).Int()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("loginlockout IsLocked: %w", err)
	}

	return n >= t.max, nil
}

func (t *Tracker) ClearFailed(ctx context.Context, email string) error {
	if email == "" {
		return nil
	}

	key := cache.KeyFailedLoginPrefix + email

	err := t.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("loginlockout ClearFailed: %w", err)
	}

	return nil
}

// RecordFailed increments the failed attempt count for the email and sets TTL on first failure.
func (t *Tracker) RecordFailed(ctx context.Context, email string) error {
	if email == "" {
		return nil
	}

	key := cache.KeyFailedLoginPrefix + email
	pipe := t.client.Pipeline()
	pipe.Incr(ctx, key)
	pipe.TTL(ctx, key)

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("loginlockout RecordFailed: %w", err)
	}

	durCmd, ok := cmds[1].(*redis.DurationCmd)
	if !ok {
		return fmt.Errorf("loginlockout RecordFailed: unexpected command type")
	}

	if err := durCmd.Err(); err != nil {
		return fmt.Errorf("loginlockout RecordFailed TTL: %w", err)
	}

	ttl := durCmd.Val()
	if ttl <= 0 {
		t.client.Expire(ctx, key, t.ttl)
	}

	return nil
}
