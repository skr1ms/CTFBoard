package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func New(host, port, password string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%s", host, port),
		Password:    password,
		DB:          0,
		DialTimeout: 5 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return rdb, nil
}
