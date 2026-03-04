package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// KeyValueStore abstracts Redis get/set/del for cache layers.
// Implementations can use *redis.Client or in-memory stores for tests.
type KeyValueStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// RedisKeyValueStore adapts *redis.Client to KeyValueStore.
type RedisKeyValueStore struct {
	Client *redis.Client
}

var _ KeyValueStore = (*RedisKeyValueStore)(nil)

func (r *RedisKeyValueStore) Get(ctx context.Context, key string) (string, error) {
	if r == nil || r.Client == nil {
		return "", redis.Nil
	}
	return r.Client.Get(ctx, key).Result()
}

func (r *RedisKeyValueStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if r == nil || r.Client == nil {
		return nil
	}
	return r.Client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisKeyValueStore) Del(ctx context.Context, keys ...string) error {
	if r == nil || r.Client == nil || len(keys) == 0 {
		return nil
	}
	return r.Client.Del(ctx, keys...).Err()
}
