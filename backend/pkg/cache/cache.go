package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

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

	v, err, _ := c.sf.Do(key, func() (any, error) {
		data, err := loadFn()
		if err != nil {
			return nil, err
		}
		if bytes, err := json.Marshal(data); err == nil {
			c.redis.Set(context.Background(), key, bytes, ttl)
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

func (c *Cache) Del(ctx context.Context, keys ...string) {
	if len(keys) > 0 {
		c.redis.Del(ctx, keys...)
	}
}

func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, bytes, ttl).Err()
}
