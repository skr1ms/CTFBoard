package helper

import (
	"net/http"
	"sync"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

var (
	dynamicRateLimitRedisErrors     *prometheus.CounterVec
	dynamicRateLimitRedisErrorsOnce sync.Once
)

func initDynamicRateLimitMetrics() {
	dynamicRateLimitRedisErrorsOnce.Do(func() {
		dynamicRateLimitRedisErrors = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rate_limit_dynamic_redis_errors_total",
				Help: "Total number of Redis errors encountered by the dynamic (config-driven) rate-limit middleware.",
			},
			[]string{"limiter"},
		)
		prometheus.MustRegister(dynamicRateLimitRedisErrors)
	})
}

const (
	dynamicLimitKeyPrefix     = "limiter:dynamic:"
	dynamicLimitFallback      = 10
	memLimiterInitialCapacity = 128
	memLimiterMaxKeys         = 10000
)

type memRateLimitEntry struct {
	count   int64
	expires time.Time
}

type memRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*memRateLimitEntry
	maxKeys int
}

func newMemRateLimiter() *memRateLimiter {
	return &memRateLimiter{
		entries: make(map[string]*memRateLimitEntry, memLimiterInitialCapacity),
		maxKeys: memLimiterMaxKeys,
	}
}

func (m *memRateLimiter) incr(key string, window time.Duration) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	e, ok := m.entries[key]
	if !ok || now.After(e.expires) {
		if !ok && len(m.entries) >= m.maxKeys {
			return 0
		}
		e = &memRateLimitEntry{count: 1, expires: now.Add(window)}
		m.entries[key] = e
		return 1
	}
	e.count++
	return e.count
}

var rateLimitIncrScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`)

//nolint:gocognit // config branches and key extraction
func RateLimitFromConfig(
	client *redis.Client,
	keyPrefix string,
	window time.Duration,
	cache *RateLimitConfigCache,
	getter SettingsGetter,
	getLimit func(*RateLimitConfig) int64,
	keyFunc func(*http.Request) (string, error),
	log logger.Logger,
) func(next http.Handler) http.Handler {
	initDynamicRateLimitMetrics()
	windowSecs := int64(window.Seconds())
	memLimiter := newMemRateLimiter()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cfg, err := cache.Get(ctx, getter)
			if err != nil {
				log.WithError(err).Warn("rate limit config cache get failed, using stale config")
				cfg = cache.GetStale()
				if cfg == nil {
					// Conservative cold-start defaults: no stale config yet (e.g. DB/Redis down
					// at startup). Use tight limits so critical paths like login are not
					// wide-open during an outage window.
					cfg = &RateLimitConfig{
						LoginPerMinute:          5,
						RegisterPerMinute:       3,
						ForgotPasswordPerMinute: defaultForgotPasswordPerMin,
						ResetPasswordPerMinute:  defaultResetPasswordPerMin,
						ScoreboardPerMinute:     defaultScoreboardPerMin,
						GeneralIPPerMinute:      defaultGeneralIPPerMin,
					}
				}
			}
			limit := getLimit(cfg)
			if limit <= 0 {
				limit = dynamicLimitFallback
			}

			key, err := keyFunc(r)
			if err != nil || key == "" {
				key = GetClientIP(r, nil)
			}
			redisKey := dynamicLimitKeyPrefix + keyPrefix + ":" + key

			count, err := rateLimitIncrScript.Run(ctx, client, []string{redisKey}, windowSecs).Int64()
			if err != nil {
				log.WithError(err).Error("rate limit redis incr failed; falling back to in-memory limiter")
				dynamicRateLimitRedisErrors.WithLabelValues(keyPrefix).Inc()

				memCount := memLimiter.incr(key, window)
				// Fail-closed: memCount == 0 means the in-memory limiter is at capacity
				// and cannot track this key. Allow the request through only if we can
				// confirm it is within the limit.
				if memCount == 0 || memCount > limit {
					HandleError(w, r, ErrTooManyRequests)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if count > limit {
				HandleError(w, r, ErrTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
