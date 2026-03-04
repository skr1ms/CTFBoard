package helper

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMemRateLimiter_FirstRequest_ReturnsOne(t *testing.T) {
	t.Parallel()
	m := newMemRateLimiter()
	count := m.incr("key1", time.Minute)
	assert.Equal(t, int64(1), count)
}

func TestMemRateLimiter_IncrementalCounts(t *testing.T) {
	t.Parallel()
	m := newMemRateLimiter()
	for i := 1; i <= 5; i++ {
		count := m.incr("key1", time.Minute)
		assert.Equal(t, int64(i), count)
	}
}

func TestMemRateLimiter_WindowExpiry_ResetsCount(t *testing.T) {
	t.Parallel()
	m := newMemRateLimiter()
	_ = m.incr("expkey", 1*time.Millisecond)
	_ = m.incr("expkey", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	count := m.incr("expkey", time.Minute)
	assert.Equal(t, int64(1), count, "counter should reset after window expiry")
}

func TestMemRateLimiter_MaxKeys_ReturnZeroOnOverflow(t *testing.T) {
	t.Parallel()
	m := &memRateLimiter{
		entries: make(map[string]*memRateLimitEntry, 2),
		maxKeys: 2,
	}
	_ = m.incr("k1", time.Minute)
	_ = m.incr("k2", time.Minute)
	count := m.incr("k3", time.Minute)
	assert.Equal(t, int64(0), count)
}

func TestMemRateLimiter_DifferentKeys_IndependentCounters(t *testing.T) {
	t.Parallel()
	m := newMemRateLimiter()
	m.incr("a", time.Minute)
	m.incr("a", time.Minute)
	m.incr("b", time.Minute)
	assert.Equal(t, int64(3), m.incr("a", time.Minute))
	assert.Equal(t, int64(2), m.incr("b", time.Minute))
}

func buildRateLimitHandler(t *testing.T, limit int64) http.Handler {
	t.Helper()
	db, redisMock := redismock.NewClientMock()
	getter := mocks.NewMockSettingsGetter(t)
	getter.EXPECT().Get(mock.Anything).Return(&entity.Settings{
		SubmitLimitPerUser:     1000000,
		SubmitLimitDurationMin: 1,
	}, nil).Maybe()

	for range 100 {
		redisMock.ExpectEvalSha("", []string{}, int64(60)).SetErr(errors.New("redis down"))
	}
	t.Cleanup(func() { redisMock.ClearExpect() })

	cache := NewRateLimitConfigCache(time.Minute)
	return RateLimitFromConfig(
		db, "test", time.Minute, cache, getter,
		func(_ *RateLimitConfig) int64 { return limit },
		func(_ *http.Request) (string, error) { return "key1", nil },
		logger.Noop(),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func TestRateLimitFromConfig_UnderLimit_Passes(t *testing.T) {
	t.Parallel()
	handler := buildRateLimitHandler(t, 100)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRateLimitFromConfig_OverLimit_Returns429(t *testing.T) {
	t.Parallel()
	const limit = 3
	handler := buildRateLimitHandler(t, limit)

	for i := 1; i <= limit; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass", i)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "request over limit should be rejected")
}

func TestRateLimitFromConfig_RedisError_FallsBackToInMemory(t *testing.T) {
	t.Parallel()
	handler := buildRateLimitHandler(t, 100)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRateLimitFromConfig_ConfigGetterError_UsesFallbackDefaults(t *testing.T) {
	t.Parallel()
	db, redisMock := redismock.NewClientMock()
	getter := mocks.NewMockSettingsGetter(t)
	getter.EXPECT().Get(mock.Anything).Return(nil, errors.New("db error")).Maybe()

	for range 5 {
		redisMock.ExpectEvalSha("", []string{}, int64(60)).SetErr(errors.New("redis down"))
	}
	t.Cleanup(func() { redisMock.ClearExpect() })

	cache := NewRateLimitConfigCache(time.Minute)
	handler := RateLimitFromConfig(
		db, "submit", time.Minute, cache, getter,
		func(c *RateLimitConfig) int64 { return int64(c.LoginPerMinute) },
		func(_ *http.Request) (string, error) { return "user1", nil },
		logger.Noop(),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRateLimitConfigCache_CachesResult(t *testing.T) {
	t.Parallel()
	getter := mocks.NewMockSettingsGetter(t)
	settings := &entity.Settings{SubmitLimitPerUser: 10, SubmitLimitDurationMin: 1}
	getter.EXPECT().Get(mock.Anything).Return(settings, nil).Once()

	cache := NewRateLimitConfigCache(time.Minute)

	cfg1, err := cache.Get(context.Background(), getter)
	require.NoError(t, err)
	cfg2, err := cache.Get(context.Background(), getter)
	require.NoError(t, err)
	assert.Equal(t, cfg1, cfg2)
}

func TestRateLimitConfigCache_GetStale_ReturnsNilWhenEmpty(t *testing.T) {
	t.Parallel()
	cache := NewRateLimitConfigCache(time.Minute)
	assert.Nil(t, cache.GetStale())
}

func TestRateLimitConfigCache_GetStale_ReturnsLastGoodConfig(t *testing.T) {
	t.Parallel()
	getter := mocks.NewMockSettingsGetter(t)
	getter.EXPECT().Get(mock.Anything).Return(&entity.Settings{
		RateLimitLoginPerMinute: 42,
	}, nil).Once()

	cache := NewRateLimitConfigCache(time.Minute)
	cfg, err := cache.Get(context.Background(), getter)
	require.NoError(t, err)

	stale := cache.GetStale()
	require.NotNil(t, stale)
	assert.Equal(t, cfg.LoginPerMinute, stale.LoginPerMinute)
}
