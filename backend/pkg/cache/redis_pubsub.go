package cache

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

var ErrRedisNotConfigured = errors.New("redis client not configured")

type RedisPubSubStore struct {
	Client *redis.Client
}

var _ PubSubStore = (*RedisPubSubStore)(nil)

func (r *RedisPubSubStore) Publish(ctx context.Context, channel, message string) error {
	if r == nil || r.Client == nil {
		return ErrRedisNotConfigured
	}
	return r.Client.Publish(ctx, channel, message).Err()
}

func (r *RedisPubSubStore) Subscribe(ctx context.Context, channel string) (<-chan string, error) {
	if r == nil || r.Client == nil {
		return nil, ErrRedisNotConfigured
	}
	out := make(chan string)
	pubsub := r.Client.Subscribe(ctx, channel)
	go func() {
		defer close(out)
		defer pubsub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-pubsub.Channel():
				if !ok {
					return
				}
				select {
				case out <- msg.Payload:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}
