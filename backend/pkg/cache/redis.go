package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/TakuyaYagam1/AstroCTFb/config"
)

const (
	redisDialTimeout  = 5 * time.Second
	redisReadTimeout  = 3 * time.Second
	redisWriteTimeout = 3 * time.Second
	redisPingTimeout  = 5 * time.Second
	redisDefaultPool  = 50
	redisDefaultIdle  = 10
)

func NewRedisClient(cfg *config.Redis) (*redis.Client, error) {
	poolSize := cfg.PoolSize
	if poolSize <= 0 {
		poolSize = redisDefaultPool
	}
	minIdle := cfg.MinIdleConns
	if minIdle <= 0 {
		minIdle = redisDefaultIdle
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           0,
		DialTimeout:  redisDialTimeout,
		ReadTimeout:  redisReadTimeout,
		WriteTimeout: redisWriteTimeout,
		PoolSize:     poolSize,
		MinIdleConns: minIdle,
	})

	ctx, cancel := context.WithTimeout(context.Background(), redisPingTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return rdb, nil
}
