package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

func startRedisForTest(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

	c, err := redisContainer.Run(ctx, "redis:alpine")
	require.NoError(t, err, "start redis container")
	t.Cleanup(func() {
		if err := c.Terminate(ctx); err != nil {
			t.Logf("terminate redis container: %v", err)
		}
	})

	uri, err := c.ConnectionString(ctx)
	require.NoError(t, err)

	opts, err := redis.ParseURL(uri)
	require.NoError(t, err)

	client := redis.NewClient(opts)
	require.NoError(t, client.Ping(ctx).Err())
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestRateLimit_UnderLimit_Passes(t *testing.T) {
	t.Parallel()
	client := startRedisForTest(t)
	l := logger.New(&logger.Options{Level: logger.ErrorLevel, Output: logger.ConsoleOutput})

	keyPrefix := fmt.Sprintf("test-under-%d", time.Now().UnixNano())

	r := chi.NewRouter()
	r.Use(RateLimit(client, keyPrefix, 10, time.Minute, func(_ *http.Request) (string, error) {
		return "test-key", nil
	}, nil, l))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass", i+1)
	}
}

func TestRateLimit_OverLimit_Returns429(t *testing.T) {
	t.Parallel()
	client := startRedisForTest(t)
	l := logger.New(&logger.Options{Level: logger.ErrorLevel, Output: logger.ConsoleOutput})

	keyPrefix := fmt.Sprintf("test-over-%d", time.Now().UnixNano())

	r := chi.NewRouter()
	r.Use(RateLimit(client, keyPrefix, 3, time.Minute, func(_ *http.Request) (string, error) {
		return "test-key", nil
	}, nil, l))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass", i+1)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "4th request should be rate limited")
}

func TestCombinedRateLimit_PassesAndSetsMinHeaders(t *testing.T) {
	t.Parallel()
	client := startRedisForTest(t)
	l := logger.New(&logger.Options{Level: logger.ErrorLevel, Output: logger.ConsoleOutput})
	keyPrefix1 := fmt.Sprintf("combined-a-%d", time.Now().UnixNano())
	keyPrefix2 := fmt.Sprintf("combined-b-%d", time.Now().UnixNano())

	specs := []RateLimitSpec{
		{KeyPrefix: keyPrefix1, Limit: 10, Window: time.Minute, KeyFunc: func(_ *http.Request) (string, error) { return "k", nil }},
		{KeyPrefix: keyPrefix2, Limit: 5, Window: time.Minute, KeyFunc: func(_ *http.Request) (string, error) { return "k", nil }},
	}
	r := chi.NewRouter()
	r.Use(CombinedRateLimit(client, specs, nil, l))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "5", rr.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "4", rr.Header().Get("X-RateLimit-Remaining"))
}

func TestPerKeyRateLimiter_Check(t *testing.T) {
	t.Parallel()
	client := startRedisForTest(t)
	keyPrefix := fmt.Sprintf("perkey-%d", time.Now().UnixNano())
	limiter, err := NewPerKeyRateLimiter(client, keyPrefix, 2, time.Minute)
	require.NoError(t, err)

	ctx := context.Background()
	ok, err := limiter.Check(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = limiter.Check(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = limiter.Check(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestRateLimit_InMemoryFallback_WhenRedisUnavailable(t *testing.T) {
	t.Parallel()
	client := startRedisForTest(t)
	l := logger.New(&logger.Options{Level: logger.ErrorLevel, Output: logger.ConsoleOutput})
	keyPrefix := fmt.Sprintf("fallback-%d", time.Now().UnixNano())

	r := chi.NewRouter()
	r.Use(RateLimit(client, keyPrefix, 2, time.Minute, func(_ *http.Request) (string, error) {
		return "fallback-key", nil
	}, nil, l))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	require.NoError(t, client.Close())

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusOK, rr.Code)

	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusOK, rr2.Code)

	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusTooManyRequests, rr3.Code)
}

func TestCombinedMemLimiter_IncrAtMaxKeys_EvictsOldestAndAcceptsNew(t *testing.T) {
	t.Parallel()
	m := newCombinedMemLimiterWithMaxKeys(2)
	window := time.Minute

	assert.Equal(t, int64(1), m.incr("a", window))
	assert.Equal(t, int64(1), m.incr("b", window))
	assert.Equal(t, int64(1), m.incr("c", window))
}
