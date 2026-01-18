package redis

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCircuitOpen is returned when the circuit breaker is open
var ErrCircuitOpen = errors.New("redis circuit breaker is open")

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

// allowRequest checks if the request should be allowed to proceed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == StateOpen {
		// If timeout passed, allow a trial request (Half-Open logic)
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			return true
		}
		return false // Still open, block request
	}

	return true
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

	// If it was Open and succeeded -> Close it
	// If it was Closed -> Reset failures just in case
	cb.failures = 0
	cb.state = StateClosed
}

func (cb *CircuitBreaker) Get(ctx context.Context, key string) *redis.StringCmd {
	if !cb.allowRequest() {
		cmd := redis.NewStringCmd(ctx)
		cmd.SetErr(ErrCircuitOpen)
		return cmd
	}

	result := cb.client.Get(ctx, key)
	// redis.Nil is NOT a system failure, it's a valid "not found" response
	if result.Err() != nil && result.Err() != redis.Nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	return result
}

func (cb *CircuitBreaker) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	if !cb.allowRequest() {
		cmd := redis.NewStatusCmd(ctx)
		cmd.SetErr(ErrCircuitOpen)
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
	if !cb.allowRequest() {
		cmd := redis.NewIntCmd(ctx)
		cmd.SetErr(ErrCircuitOpen)
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
	// Ping also uses circuit breaker to detect recovery
	if !cb.allowRequest() {
		cmd := redis.NewStatusCmd(ctx)
		cmd.SetErr(ErrCircuitOpen)
		return cmd
	}

	result := cb.client.Ping(ctx)
	if result.Err() != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	return result
}

func (cb *CircuitBreaker) Close() error {
	return cb.client.Close()
}

func (cb *CircuitBreaker) Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
	if !cb.allowRequest() {
		cmd := redis.NewIntCmd(ctx)
		cmd.SetErr(ErrCircuitOpen)
		return cmd
	}

	result := cb.client.Publish(ctx, channel, message)
	if result.Err() != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	return result
}

func (cb *CircuitBreaker) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	// Subscribe is a long-lived connection, generally bypasses CB logic
	// But calls will eventually fail if connection is down
	return cb.client.Subscribe(ctx, channels...)
}

func (cb *CircuitBreaker) Incr(ctx context.Context, key string) *redis.IntCmd {
	if !cb.allowRequest() {
		cmd := redis.NewIntCmd(ctx)
		cmd.SetErr(ErrCircuitOpen)
		return cmd
	}

	result := cb.client.Incr(ctx, key)
	if result.Err() != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	return result
}

func (cb *CircuitBreaker) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	if !cb.allowRequest() {
		cmd := redis.NewBoolCmd(ctx)
		cmd.SetErr(ErrCircuitOpen)
		return cmd
	}

	result := cb.client.Expire(ctx, key, expiration)
	if result.Err() != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	return result
}
