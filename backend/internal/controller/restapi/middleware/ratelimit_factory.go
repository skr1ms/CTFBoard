package middleware

import (
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/httprate"
	httprateredis "github.com/go-chi/httprate-redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
)

var rateLimitRedisErrors = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rate_limit_redis_errors_total",
		Help: "Total number of Redis errors encountered by rate-limit middleware.",
	},
	[]string{rateLimitMetricLabelLimiter},
)

func fallbackKeyFunc(keyFunc func(r *http.Request) (string, error)) func(r *http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		k, err := keyFunc(r)
		if err != nil || k == "" {
			return kitMiddleware.GetClientIPFromContext(r.Context()), nil
		}

		return k, nil
	}
}

func newRedisLimitCounter(client *redis.Client, prefixKey, label, message string, log logkit.Logger) (httprate.LimitCounter, error) {
	return httprateredis.NewRedisLimitCounter(&httprateredis.Config{
		Client:    client,
		PrefixKey: prefixKey,
		OnError: func(err error) {
			if log != nil {
				log.WithError(err).WithFields(logkit.Fields{rateLimitLogFieldKeyPrefix: label}).Warn(message)
			}

			rateLimitRedisErrors.WithLabelValues(label).Inc()
		},
	})
}

func tooManyRequestsHandler(onRateLimited func(*http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if onRateLimited != nil {
			onRateLimited(r)
		}

		httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrTooManyRequests))
	}
}

func rateLimitConfigOrFallback(ctxLog logkit.Logger, r *http.Request, cache *RateLimitConfigCache, getter SettingsGetter) *RateLimitConfig {
	cfg, err := cache.Get(r.Context(), getter)
	if err == nil {
		return cfg
	}

	ctxLog.WithError(err).Warn("rate limit config cache get failed, using stale config")

	cfg = cache.GetStale()
	if cfg == nil {
		return defaultFallbackConfig
	}

	return cfg
}

// RateLimit returns a Redis-backed rate-limit middleware for the given key prefix and limit.
// Falls back to an in-memory counter if Redis is unavailable; errors are counted in prometheus.
// An empty or errored keyFunc result falls back to the client IP.
func RateLimit(client *redis.Client, keyPrefix string, limit int64, window time.Duration, keyFunc func(r *http.Request) (string, error), log logkit.Logger) func(next http.Handler) http.Handler {
	counter, err := newRedisLimitCounter(
		client,
		cache.KeyLimiterPrefix+keyPrefix,
		keyPrefix,
		"middleware - RateLimit: Redis error, using in-memory fallback",
		log,
	)
	if err != nil {
		log.WithError(err).Fatal("failed to create rate limit store")

		return nil
	}

	return httprate.Limit(int(limit), window,
		httprate.WithKeyFuncs(fallbackKeyFunc(keyFunc)),
		httprate.WithLimitCounter(counter),
		httprate.WithLimitHandler(tooManyRequestsHandler(nil)),
	)
}

// DynamicRateLimit returns a rate-limit middleware whose limit is read from RateLimitConfigCache
// on every request, allowing live updates without a server restart.
// Falls back to DynamicLimitFallback if the cache returns an error or a zero limit.
func DynamicRateLimit(
	client *redis.Client,
	keyPrefix string,
	window time.Duration,
	cache *RateLimitConfigCache,
	getter SettingsGetter,
	getLimit func(*RateLimitConfig) int64,
	keyFunc func(*http.Request) (string, error),
	log logkit.Logger,
) func(next http.Handler) http.Handler {
	counter, err := newRedisLimitCounter(
		client,
		DynamicLimitKeyPrefix+keyPrefix,
		keyPrefix,
		"dynamic rate limit: Redis error, using in-memory fallback",
		log,
	)
	if err != nil {
		log.WithError(err).Fatal("failed to create dynamic rate limit store")

		return nil
	}

	limiter := httprate.NewRateLimiter(int(DynamicLimitFallback), window,
		httprate.WithKeyFuncs(fallbackKeyFunc(keyFunc)),
		httprate.WithLimitCounter(counter),
		httprate.WithLimitHandler(tooManyRequestsHandler(nil)),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := rateLimitConfigOrFallback(log, r, cache, getter)

			limit := getLimit(cfg)
			if limit <= 0 {
				limit = DynamicLimitFallback
			}

			r = r.WithContext(httprate.WithRequestLimit(r.Context(), int(limit)))
			limiter.Handler(next).ServeHTTP(w, r)
		})
	}
}

// RateLimitSpec describes a single limiter layer used by CombinedRateLimit.
type RateLimitSpec struct {
	KeyPrefix string
	Limit     int64
	Window    time.Duration
	KeyFunc   func(r *http.Request) (string, error)
}

// CombinedRateLimit stacks multiple Redis-backed rate limiters into a single middleware chain.
// All specs are applied in order; the first limiter to fire returns 429.
func CombinedRateLimit(client *redis.Client, specs []RateLimitSpec, log logkit.Logger) func(next http.Handler) http.Handler {
	mws := make([]func(http.Handler) http.Handler, 0, len(specs))
	for _, s := range specs {
		counter, err := newRedisLimitCounter(
			client,
			cache.KeyLimiterPrefix+s.KeyPrefix,
			s.KeyPrefix,
			"middleware - CombinedRateLimit: Redis error, using in-memory fallback",
			log,
		)
		if err != nil {
			log.WithError(err).WithFields(logkit.Fields{rateLimitLogFieldKeyPrefix: s.KeyPrefix}).Fatal("middleware - CombinedRateLimit: failed to create store")

			return nil
		}

		mw := httprate.Limit(int(s.Limit), s.Window,
			httprate.WithKeyFuncs(fallbackKeyFunc(s.KeyFunc)),
			httprate.WithLimitCounter(counter),
			httprate.WithLimitHandler(tooManyRequestsHandler(nil)),
		)
		mws = append(mws, mw)
	}

	return func(next http.Handler) http.Handler {
		h := next

		for _, mw := range slices.Backward(mws) {
			h = mw(h)
		}

		return h
	}
}
