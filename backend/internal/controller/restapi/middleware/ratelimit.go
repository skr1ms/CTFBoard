package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
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

func RateLimit(client *redis.Client, keyPrefix string, limit int64, window time.Duration, keyFunc func(r *http.Request) (string, error), trustedProxyCIDRs []string, logger logger.Logger) func(next http.Handler) http.Handler {
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
	memFallback := newCombinedMemLimiter()
	initRateLimitMetrics()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, err := keyFunc(r)
			if err != nil || key == "" {
				key = httputil.GetClientIP(r, trustedProxyCIDRs)
			}
			lctx, lerr := instance.Get(r.Context(), key)
			if lerr != nil {
				if logger != nil {
					logger.WithError(lerr).WithFields(map[string]any{"key_prefix": keyPrefix}).Warn("middleware - RateLimit: Redis error, using in-memory fallback")
				}
				rateLimitRedisErrors.WithLabelValues(keyPrefix).Inc()
				memCount := memFallback.incr(key, window)
				if memCount > limit {
					httputil.HandleError(w, r, httperr.ErrTooManyRequests)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			if lctx.Reached {
				w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(lctx.Limit, 10))
				w.Header().Set("X-RateLimit-Remaining", "0")
				httputil.HandleError(w, r, httperr.ErrTooManyRequests)
				return
			}
			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(lctx.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(lctx.Remaining, 10))
			next.ServeHTTP(w, r)
		})
	}
}

type RateLimitSpec struct {
	KeyPrefix string
	Limit     int64
	Window    time.Duration
	KeyFunc   func(r *http.Request) (string, error)
}

const (
	combinedMemLimiterMaxKeys = 10000
	combinedMemLimiterCap     = 128
)

type combinedMemEntry struct {
	count   int64
	expires time.Time
}

type combinedMemLimiter struct {
	mu      sync.Mutex
	entries map[string]*combinedMemEntry
	maxKeys int
}

func newCombinedMemLimiter() *combinedMemLimiter {
	return newCombinedMemLimiterWithMaxKeys(combinedMemLimiterMaxKeys)
}

func newCombinedMemLimiterWithMaxKeys(maxKeys int) *combinedMemLimiter {
	return &combinedMemLimiter{
		entries: make(map[string]*combinedMemEntry, combinedMemLimiterCap),
		maxKeys: maxKeys,
	}
}

func (m *combinedMemLimiter) purgeStale() {
	now := time.Now()
	for k, e := range m.entries {
		if now.After(e.expires) {
			delete(m.entries, k)
		}
	}
}

func (m *combinedMemLimiter) evictOldest() {
	var oldestKey string
	var oldestExpires time.Time
	first := true
	for k, e := range m.entries {
		if first || e.expires.Before(oldestExpires) {
			oldestKey = k
			oldestExpires = e.expires
			first = false
		}
	}
	if oldestKey != "" {
		delete(m.entries, oldestKey)
	}
}

func (m *combinedMemLimiter) incr(key string, window time.Duration) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	e, ok := m.entries[key]
	if !ok || now.After(e.expires) {
		if !ok {
			if len(m.entries) >= m.maxKeys {
				m.purgeStale()
				if len(m.entries) >= m.maxKeys {
					m.evictOldest()
				}
			}
		}
		e = &combinedMemEntry{count: 1, expires: now.Add(window)}
		m.entries[key] = e
		return 1
	}
	e.count++
	return e.count
}

func CombinedRateLimit(client *redis.Client, specs []RateLimitSpec, trustedProxyCIDRs []string, log logger.Logger) func(next http.Handler) http.Handler {
	type entry struct {
		instance   *limiter.Limiter
		keyFunc    func(r *http.Request) (string, error)
		memLimiter *combinedMemLimiter
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
			instance:   limiter.New(store, limiter.Rate{Period: s.Window, Limit: s.Limit}),
			keyFunc:    s.KeyFunc,
			memLimiter: newCombinedMemLimiter(),
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
						key = httputil.GetClientIP(r, trustedProxyCIDRs)
					}
					lctx, lerr := e.instance.Get(r.Context(), key)
					results[i] = checkResult{lctx: lctx, err: lerr}
				}()
			}
			wg.Wait()

			var minLimit, minRemaining int64 = -1, -1
			for i, res := range results {
				if res.err != nil {
					log.WithError(res.err).WithFields(logger.Fields{"key_prefix": specs[i].KeyPrefix}).Error("middleware - CombinedRateLimit: Redis error, using in-memory fallback")
					rateLimitRedisErrors.WithLabelValues(specs[i].KeyPrefix).Inc()
					key, kerr := entries[i].keyFunc(r)
					if kerr != nil || key == "" {
						key = httputil.GetClientIP(r, trustedProxyCIDRs)
					}
					memCount := entries[i].memLimiter.incr(key, specs[i].Window)
					if memCount > specs[i].Limit {
						httputil.HandleError(w, r, httperr.ErrTooManyRequests)
						return
					}
					if minLimit < 0 || specs[i].Limit < minLimit {
						minLimit = specs[i].Limit
					}
					remaining := specs[i].Limit - memCount
					if remaining < 0 {
						remaining = 0
					}
					if minRemaining < 0 || remaining < minRemaining {
						minRemaining = remaining
					}
					continue
				}
				if res.lctx.Reached {
					w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(res.lctx.Limit, 10))
					w.Header().Set("X-RateLimit-Remaining", "0")
					httputil.HandleError(w, r, httperr.ErrTooManyRequests)
					return
				}
				if minLimit < 0 || res.lctx.Limit < minLimit {
					minLimit = res.lctx.Limit
				}
				if minRemaining < 0 || res.lctx.Remaining < minRemaining {
					minRemaining = res.lctx.Remaining
				}
			}
			if minLimit >= 0 {
				w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(minLimit, 10))
				w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(minRemaining, 10))
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
