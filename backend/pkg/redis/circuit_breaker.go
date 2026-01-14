package redis

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	client           *redis.Client
	state            CircuitState
	failures         int
	failureThreshold int
	resetTimeout     time.Duration
	lastFailure      time.Time
	mu               sync.RWMutex
}

func NewCircuitBreaker(client *redis.Client, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		client:           client,
		state:            StateClosed,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
	}
}

func (cb *CircuitBreaker) isOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return time.Since(cb.lastFailure) <= cb.resetTimeout
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = StateClosed
}

func (cb *CircuitBreaker) Get(ctx context.Context, key string) *redis.StringCmd {
	if cb.isOpen() {
		cmd := redis.NewStringCmd(ctx)
		cmd.SetErr(redis.Nil)
		return cmd
	}

	result := cb.client.Get(ctx, key)
	if result.Err() != nil && result.Err() != redis.Nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	return result
}

func (cb *CircuitBreaker) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	if cb.isOpen() {
		cmd := redis.NewStatusCmd(ctx)
		return cmd
	}

	result := cb.client.Set(ctx, key, value, expiration)
	if result.Err() != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	return result
}

func (cb *CircuitBreaker) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	if cb.isOpen() {
		cmd := redis.NewIntCmd(ctx)
		return cmd
	}

	result := cb.client.Del(ctx, keys...)
	if result.Err() != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	return result
}

func (cb *CircuitBreaker) Ping(ctx context.Context) *redis.StatusCmd {
	return cb.client.Ping(ctx)
}

func (cb *CircuitBreaker) Close() error {
	return cb.client.Close()
}
