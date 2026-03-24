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
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/testutil"
)

type staticSettingsGetter struct {
	settings *domain.Settings
}

func (g *staticSettingsGetter) Get(_ context.Context) (*domain.Settings, error) {
	return g.settings, nil
}

func startRedisForTest(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()
	client, cleanup, err := testutil.StartRedisClient(ctx)
	require.NoError(t, err, "start redis")
	t.Cleanup(cleanup)
	return client
}

func TestRateLimit_UnderLimit_Passes(t *testing.T) {
	t.Parallel()
	client := startRedisForTest(t)
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)

	keyPrefix := fmt.Sprintf("test-under-%d", time.Now().UnixNano())

	r := chi.NewRouter()
	r.Use(RateLimit(client, keyPrefix, 10, time.Minute, func(_ *http.Request) (string, error) {
		return "test-key", nil
	}, l))
	r.Get("/", okHandler())

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
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)

	keyPrefix := fmt.Sprintf("test-over-%d", time.Now().UnixNano())

	r := chi.NewRouter()
	r.Use(RateLimit(client, keyPrefix, 3, time.Minute, func(_ *http.Request) (string, error) {
		return "test-key", nil
	}, l))
	r.Get("/", okHandler())

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
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)
	keyPrefix1 := fmt.Sprintf("combined-a-%d", time.Now().UnixNano())
	keyPrefix2 := fmt.Sprintf("combined-b-%d", time.Now().UnixNano())

	specs := []RateLimitSpec{
		{KeyPrefix: keyPrefix1, Limit: 10, Window: time.Minute, KeyFunc: func(_ *http.Request) (string, error) { return "k", nil }},
		{KeyPrefix: keyPrefix2, Limit: 5, Window: time.Minute, KeyFunc: func(_ *http.Request) (string, error) { return "k", nil }},
	}
	r := chi.NewRouter()
	r.Use(CombinedRateLimit(client, specs, l))
	r.Get("/", okHandler())

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
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)
	keyPrefix := fmt.Sprintf("fallback-%d", time.Now().UnixNano())

	r := chi.NewRouter()
	r.Use(RateLimit(client, keyPrefix, 2, time.Minute, func(_ *http.Request) (string, error) {
		return "fallback-key", nil
	}, l))
	r.Get("/", okHandler())

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

func TestDynamicRateLimit_UnderLimit_Passes(t *testing.T) {
	t.Parallel()
	client := startRedisForTest(t)
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)
	keyPrefix := fmt.Sprintf("dynamic-under-%d", time.Now().UnixNano())
	getter := &staticSettingsGetter{settings: &domain.Settings{RateLimitLoginPerMinute: 100}}
	cache := NewRateLimitConfigCache(time.Minute)

	handler := DynamicRateLimit(
		client, keyPrefix, time.Minute, cache, getter,
		func(c *RateLimitConfig) int64 { return int64(c.LoginPerMinute) },
		func(_ *http.Request) (string, error) { return "key1", nil },
		l,
	)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDynamicRateLimit_OverLimit_Returns429(t *testing.T) {
	t.Parallel()
	client := startRedisForTest(t)
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)
	keyPrefix := fmt.Sprintf("dynamic-over-%d", time.Now().UnixNano())
	getter := &staticSettingsGetter{settings: &domain.Settings{RateLimitLoginPerMinute: 3}}
	cache := NewRateLimitConfigCache(time.Minute)

	handler := DynamicRateLimit(
		client, keyPrefix, time.Minute, cache, getter,
		func(c *RateLimitConfig) int64 { return int64(c.LoginPerMinute) },
		func(_ *http.Request) (string, error) { return "key1", nil },
		l,
	)(okHandler())

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d", i+1)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestDynamicRateLimit_SetsResponseHeaders(t *testing.T) {
	t.Parallel()
	client := startRedisForTest(t)
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)
	keyPrefix := fmt.Sprintf("dynamic-headers-%d", time.Now().UnixNano())
	getter := &staticSettingsGetter{settings: &domain.Settings{RateLimitLoginPerMinute: 10}}
	cache := NewRateLimitConfigCache(time.Minute)

	handler := DynamicRateLimit(
		client, keyPrefix, time.Minute, cache, getter,
		func(c *RateLimitConfig) int64 { return int64(c.LoginPerMinute) },
		func(_ *http.Request) (string, error) { return "key1", nil },
		l,
	)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "10", rr.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Reset"))
}

func TestDynamicRateLimit_InMemoryFallback_WhenRedisUnavailable(t *testing.T) {
	t.Parallel()
	client := startRedisForTest(t)
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)
	keyPrefix := fmt.Sprintf("dynamic-fallback-%d", time.Now().UnixNano())
	getter := &staticSettingsGetter{settings: &domain.Settings{RateLimitLoginPerMinute: 2}}
	cache := NewRateLimitConfigCache(time.Minute)

	handler := DynamicRateLimit(
		client, keyPrefix, time.Minute, cache, getter,
		func(c *RateLimitConfig) int64 { return int64(c.LoginPerMinute) },
		func(_ *http.Request) (string, error) { return "fallback-key", nil },
		l,
	)(okHandler())

	require.NoError(t, client.Close())

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusOK, rr.Code)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusOK, rr2.Code)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusTooManyRequests, rr3.Code)
}

func TestRateLimitConfigCache_GetStale_ReturnsNilWhenEmpty(t *testing.T) {
	t.Parallel()
	cache := NewRateLimitConfigCache(time.Minute)
	assert.Nil(t, cache.GetStale())
}

func TestRateLimitConfigCache_GetStale_ReturnsLastGoodConfig(t *testing.T) {
	t.Parallel()
	getter := &staticSettingsGetter{settings: &domain.Settings{RateLimitLoginPerMinute: 42}}
	cache := NewRateLimitConfigCache(time.Minute)
	_, err := cache.Get(context.Background(), getter)
	require.NoError(t, err)
	stale := cache.GetStale()
	require.NotNil(t, stale)
	assert.Equal(t, 42, stale.LoginPerMinute)
}
