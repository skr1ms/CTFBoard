package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/httprate"
	httprateredis "github.com/go-chi/httprate-redis"
	"github.com/redis/go-redis/v9"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
)

type perKeyCtxKey struct{}

var (
	errPerKeyNotInContext = errors.New("per-key rate limiter: key not in context")
	errRateLimitBackend   = errors.New("rate limit backend error")
)

// PerKeyRateLimiter checks rate limits programmatically (outside HTTP middleware) using a caller-supplied key.
type PerKeyRateLimiter struct {
	limiter *httprate.RateLimiter
}

func perKeyKeyFunc(r *http.Request) (string, error) {
	k := r.Context().Value(perKeyCtxKey{})
	if s, ok := k.(string); ok {
		return s, nil
	}

	return "", errPerKeyNotInContext
}

// NewPerKeyRateLimiter creates a PerKeyRateLimiter backed by Redis with the given key prefix, limit, and window.
func NewPerKeyRateLimiter(client *redis.Client, keyPrefix string, limit int64, window time.Duration) (*PerKeyRateLimiter, error) {
	counter, err := httprateredis.NewRedisLimitCounter(&httprateredis.Config{
		Client:    client,
		PrefixKey: cache.KeyLimiterPrefix + keyPrefix,
	})
	if err != nil {
		return nil, err
	}

	limiter := httprate.NewRateLimiter(int(limit), window,
		httprate.WithKeyFuncs(perKeyKeyFunc),
		httprate.WithLimitCounter(counter),
	)

	return &PerKeyRateLimiter{limiter: limiter}, nil
}

// Check returns true if the key is within the rate limit, false if it is exceeded.
// Uses a synthetic HTTP request/response pair to invoke the underlying httprate limiter without a real handler.
func (l *PerKeyRateLimiter) Check(ctx context.Context, key string) (bool, error) {
	ctx = httprate.WithRequestLimit(ctx, 0)
	ctx = context.WithValue(ctx, perKeyCtxKey{}, key)
	r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()
	l.limiter.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, r)

	if w.Code == http.StatusTooManyRequests {
		return false, nil
	}

	if w.Code == http.StatusPreconditionRequired {
		return false, errRateLimitBackend
	}

	return true, nil
}
