package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"
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
	}, l))
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
	}, l))
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
