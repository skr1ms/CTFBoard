package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	mhttp "github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

var (
	rateLimitRedisErrors     *prometheus.CounterVec
	rateLimitRedisErrorsOnce sync.Once
)

func initRateLimitMetrics() {
	rateLimitRedisErrorsOnce.Do(func() {
		rateLimitRedisErrors = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rate_limit_redis_errors_total",
				Help: "Total number of Redis errors encountered by rate-limit middleware.",
			},
			[]string{"limiter"},
		)
		prometheus.MustRegister(rateLimitRedisErrors)
	})
}

func RateLimit(client *redis.Client, keyPrefix string, limit int64, window time.Duration, keyFunc func(r *http.Request) (string, error), logger logger.Logger) func(next http.Handler) http.Handler {
	store, err := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix:   cache.KeyLimiterPrefix + keyPrefix,
		MaxRetry: 3,
	})
	if err != nil {
		logger.WithError(err).Fatal("failed to create rate limit store")
		return nil
	}

	rate := limiter.Rate{
		Period: window,
		Limit:  limit,
	}
	instance := limiter.New(store, rate)

	middleware := mhttp.NewMiddleware(instance, mhttp.WithKeyGetter(func(r *http.Request) string {
		key, err := keyFunc(r)
		if err != nil || key == "" {
			return httputil.GetClientIP(r, nil)
		}
		return key
	}))

	return middleware.Handler
}

type RateLimitSpec struct {
	KeyPrefix string
	Limit     int64
	Window    time.Duration
	KeyFunc   func(r *http.Request) (string, error)
}

//nolint:gocognit // intentional: fan-out + fail-closed Redis error handling in one pass
func CombinedRateLimit(client *redis.Client, specs []RateLimitSpec, log logger.Logger) func(next http.Handler) http.Handler {
	type entry struct {
		instance *limiter.Limiter
		keyFunc  func(r *http.Request) (string, error)
	}

	entries := make([]entry, 0, len(specs))
	for _, s := range specs {
		store, err := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
			Prefix:   cache.KeyLimiterPrefix + s.KeyPrefix,
			MaxRetry: 3,
		})
		if err != nil {
			log.WithError(err).WithFields(logger.Fields{"key_prefix": s.KeyPrefix}).Fatal("middleware - CombinedRateLimit: failed to create store")
			return nil
		}
		entries = append(entries, entry{
			instance: limiter.New(store, limiter.Rate{Period: s.Window, Limit: s.Limit}),
			keyFunc:  s.KeyFunc,
		})
	}

	type checkResult struct {
		lctx limiter.Context
		err  error
	}

	initRateLimitMetrics()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results := make([]checkResult, len(entries))
			var wg sync.WaitGroup
			wg.Add(len(entries))

			for i, e := range entries {
				go func() {
					defer wg.Done()
					key, kerr := e.keyFunc(r)
					if kerr != nil || key == "" {
						key = httputil.GetClientIP(r, nil)
					}
					lctx, lerr := e.instance.Get(r.Context(), key)
					results[i] = checkResult{lctx: lctx, err: lerr}
				}()
			}
			wg.Wait()

			for i, res := range results {
				if res.err != nil {
					log.WithError(res.err).WithFields(logger.Fields{"key_prefix": specs[i].KeyPrefix}).Error("middleware - CombinedRateLimit: Redis error")
					rateLimitRedisErrors.WithLabelValues(specs[i].KeyPrefix).Inc()
					httputil.HandleError(w, r, httperr.New(res.err, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"))
					return
				}
				if res.lctx.Reached {
					w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(res.lctx.Limit, 10))
					w.Header().Set("X-RateLimit-Remaining", "0")
					httputil.HandleError(w, r, httperr.ErrTooManyRequests)
					return
				}
				w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(res.lctx.Limit, 10))
				w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(res.lctx.Remaining, 10))
			}

			next.ServeHTTP(w, r)
		})
	}
}

type PerKeyRateLimiter struct {
	instance *limiter.Limiter
}

func NewPerKeyRateLimiter(client *redis.Client, keyPrefix string, limit int64, window time.Duration) (*PerKeyRateLimiter, error) {
	store, err := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix:   cache.KeyLimiterPrefix + keyPrefix,
		MaxRetry: 3,
	})
	if err != nil {
		return nil, err
	}
	return &PerKeyRateLimiter{
		instance: limiter.New(store, limiter.Rate{Period: window, Limit: limit}),
	}, nil
}

// Check returns true if the request for key is within the rate limit.
func (l *PerKeyRateLimiter) Check(ctx context.Context, key string) (bool, error) {
	lctx, err := l.instance.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return !lctx.Reached, nil
}
