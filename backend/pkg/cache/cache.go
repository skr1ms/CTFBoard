package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// sfKey returns the singleflight deduplication key for a cache key.
// Using the raw cache key (without a type suffix) lets Del/Forget use the same
// key space to cancel in-flight goroutines on invalidation.
// The same cache key must always map to the same Go type - mixing types for
// the same key is a programming error regardless.
func sfKey[T any](key string) string {
	return key
}

type Cache struct {
	redis *redis.Client
	sf    singleflight.Group
}

func New(redis *redis.Client) *Cache {
	return &Cache{redis: redis}
}

func GetOrLoad[T any](c *Cache, ctx context.Context, key string, ttl time.Duration, loadFn func() (T, error)) (T, error) {
	var result T

	val, err := c.redis.Get(ctx, key).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(val), &result); err == nil {
			return result, nil
		}
	}

	v, err, _ := c.sf.Do(sfKey[T](key), func() (any, error) {
		data, err := loadFn()
		if err != nil {
			return nil, fmt.Errorf("cache get: %w", err)
		}
		if bytes, err := json.Marshal(data); err == nil {
			_ = c.redis.Set(context.WithoutCancel(ctx), key, bytes, ttl).Err() //nolint:errcheck
		}
		return data, nil
	})

	if err != nil {
		var zero T
		return zero, err
	}
	cached, ok := v.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("cache: unexpected type")
	}
	return cached, nil
}

func (c *Cache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	for _, key := range keys {
		c.sf.Forget(key)
	}
	return c.redis.Del(ctx, keys...).Err()
}

func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache set marshal: %w", err)
	}
	if err := c.redis.Set(ctx, key, bytes, ttl).Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}

const deleteByPrefixBatchSize = 100

// DeleteByPrefix deletes all keys matching prefix (prefix*). Uses SCAN to avoid blocking.
func (c *Cache) DeleteByPrefix(ctx context.Context, prefix string) error {
	match := prefix + "*"
	var cursor uint64
	for {
		keys, nextCursor, err := c.redis.Scan(ctx, cursor, match, deleteByPrefixBatchSize).Result()
		if err != nil {
			return fmt.Errorf("cache delete by prefix scan: %w", err)
		}
		if len(keys) > 0 {
			for _, key := range keys {
				c.sf.Forget(key)
			}
			if err := c.redis.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("cache delete by prefix del: %w", err)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
