package cache

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"

	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
)

func NewRedisClient(ctx context.Context, cfg *config.Redis) (*redis.Client, error) {
	port, err := strconv.Atoi(cfg.Port)
	if err != nil {
		return nil, err
	}
	return cachekit.NewRedisClient(ctx, &cachekit.RedisConfig{
		Host:         cfg.Host,
		Port:         port,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})
}
