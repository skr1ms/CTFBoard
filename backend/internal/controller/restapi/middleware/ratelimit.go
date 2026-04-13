package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	httprateredis "github.com/go-chi/httprate-redis"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/semaphore"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type perKeyCtxKey struct{}

const (
	DynamicLimitKeyPrefix       = "limiter:dynamic:"
	defaultLoginPerMin          = 10
	defaultRegisterPerMin       = 5
	defaultForgotPasswordPerMin = 3
	defaultResetPasswordPerMin  = 5
	defaultLogoutPerMin         = 10
	defaultRefreshPerMin        = 10
	defaultScoreboardPerMin     = 30
	defaultGeneralIPPerMin      = 100
	defaultVerifyEmailPerMin    = 10
	defaultOAuthCallbackPerMin  = 20
	defaultOAuthRedirectPerMin  = 20
	defaultHintUnlockMinFloor   = 30
	defaultSubmitPerUser        = 10
	defaultSubmitDurationMin    = 1
	submitIPMultiplier          = 3
	hintUnlockMultiplier        = 3
	defaultCommentPerMinute     = 30
	defaultRatingPerMinute      = 30
	DynamicLimitFallback        = 10
	ratelimitAuditTimeout       = 5 * time.Second
)

var (
	errPerKeyNotInContext = errors.New("per-key rate limiter: key not in context")
	errRateLimitBackend   = errors.New("rate limit backend error")
)

var defaultFallbackConfig = &RateLimitConfig{
	LoginPerMinute:          defaultLoginPerMin,
	RegisterPerMinute:       defaultRegisterPerMin,
	ForgotPasswordPerMinute: defaultForgotPasswordPerMin,
	ResetPasswordPerMinute:  defaultResetPasswordPerMin,
	LogoutPerMinute:         defaultLogoutPerMin,
	RefreshPerMinute:        defaultRefreshPerMin,
	ScoreboardPerMinute:     defaultScoreboardPerMin,
	GeneralIPPerMinute:      defaultGeneralIPPerMin,
	VerifyEmailPerMinute:    defaultVerifyEmailPerMin,
	OAuthCallbackPerMinute:  defaultOAuthCallbackPerMin,
	OAuthRedirectPerMinute:  defaultOAuthRedirectPerMin,
	SubmitUserPerMinute:     defaultSubmitPerUser,
	SubmitIPPerMinute:       defaultSubmitPerUser * submitIPMultiplier,
	HintUnlockUserPerMinute: defaultSubmitPerUser * hintUnlockMultiplier,
	CommentPerMinute:        defaultCommentPerMinute,
	RatingPerMinute:         defaultRatingPerMinute,
}

// SettingsGetter is the minimal interface required by rate-limit middleware to read dynamic settings.
type SettingsGetter interface {
	Get(ctx context.Context) (*domain.Settings, error)
}

// RateLimitConfig holds per-endpoint rate limits (requests per minute) derived from dynamic settings.
type RateLimitConfig struct {
	LoginPerMinute          int
	RegisterPerMinute       int
	ForgotPasswordPerMinute int
	ResetPasswordPerMinute  int
	LogoutPerMinute         int
	RefreshPerMinute        int
	ScoreboardPerMinute     int
	GeneralIPPerMinute      int
	VerifyEmailPerMinute    int
	OAuthCallbackPerMinute  int
	OAuthRedirectPerMinute  int
	SubmitUserPerMinute     int
	SubmitIPPerMinute       int
	HintUnlockUserPerMinute int
	CommentPerMinute        int
	RatingPerMinute         int
}

const rateLimitConfigCacheKey = "rate_limit_config"

// RateLimitConfigCache is a short-lived in-process cache for RateLimitConfig
// so settings are not fetched from Redis/DB on every request.
type RateLimitConfigCache struct {
	cv *cachekit.CachedValue[*RateLimitConfig]
}

// NewRateLimitConfigCache creates a RateLimitConfigCache with the given TTL.
func NewRateLimitConfigCache(ttl time.Duration) *RateLimitConfigCache {
	return &RateLimitConfigCache{
		cv: cachekit.NewCachedValue[*RateLimitConfig](context.Background(), rateLimitConfigCacheKey, ttl),
	}
}

// Get returns a cached RateLimitConfig, fetching from settings via getter on cache miss.
func (c *RateLimitConfigCache) Get(ctx context.Context, getter SettingsGetter) (*RateLimitConfig, error) {
	return c.cv.Get(ctx, func(ctx context.Context) (*RateLimitConfig, error) {
		return GetRateLimitConfig(ctx, getter)
	})
}

// GetStale returns the last successfully fetched config without triggering a refresh, or nil if none.
func (c *RateLimitConfigCache) GetStale() *RateLimitConfig {
	cfg, ok := c.cv.GetStale()
	if !ok {
		return nil
	}

	return cfg
}

// Invalidate clears the cached config so the next Get fetches fresh values from settings.
func (c *RateLimitConfigCache) Invalidate() {
	c.cv.Invalidate()
}

func rateLimitDefault(value, defaultValue int) int {
	if v := max(value, 0); v != 0 {
		return v
	}

	return defaultValue
}

// GetRateLimitConfig builds a RateLimitConfig from dynamic settings, applying hardcoded defaults
// for any zero or missing values. Submit limits are derived from the per-user quota and duration.
func GetRateLimitConfig(ctx context.Context, getter SettingsGetter) (*RateLimitConfig, error) {
	settings, err := getter.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("RateLimitConfigCache - Get: %w", err)
	}

	submitUser := rateLimitDefault(settings.SubmitLimitPerUser, defaultSubmitPerUser)
	durMin := rateLimitDefault(settings.SubmitLimitDurationMin, defaultSubmitDurationMin)

	submitUserPerMin := submitUser / durMin
	if submitUserPerMin <= 0 {
		submitUserPerMin = 1
	}

	hintUnlockUserPerMin := max(submitUserPerMin*hintUnlockMultiplier, defaultHintUnlockMinFloor)

	return &RateLimitConfig{
		LoginPerMinute:          rateLimitDefault(settings.RateLimitLoginPerMinute, defaultLoginPerMin),
		RegisterPerMinute:       rateLimitDefault(settings.RateLimitRegisterPerMinute, defaultRegisterPerMin),
		ForgotPasswordPerMinute: rateLimitDefault(settings.RateLimitForgotPasswordPerMinute, defaultForgotPasswordPerMin),
		ResetPasswordPerMinute:  rateLimitDefault(settings.RateLimitResetPasswordPerMinute, defaultResetPasswordPerMin),
		LogoutPerMinute:         rateLimitDefault(settings.RateLimitLogoutPerMinute, defaultLogoutPerMin),
		RefreshPerMinute:        rateLimitDefault(settings.RateLimitRefreshPerMinute, defaultRefreshPerMin),
		ScoreboardPerMinute:     rateLimitDefault(settings.RateLimitScoreboardPerMinute, defaultScoreboardPerMin),
		GeneralIPPerMinute:      rateLimitDefault(settings.RateLimitGeneralIPPerMinute, defaultGeneralIPPerMin),
		VerifyEmailPerMinute:    rateLimitDefault(settings.RateLimitVerifyEmailPerMinute, defaultVerifyEmailPerMin),
		OAuthCallbackPerMinute:  rateLimitDefault(settings.RateLimitOAuthCallbackPerMinute, defaultOAuthCallbackPerMin),
		OAuthRedirectPerMinute:  rateLimitDefault(settings.RateLimitOAuthRedirectPerMinute, defaultOAuthRedirectPerMin),
		SubmitUserPerMinute:     submitUserPerMin,
		SubmitIPPerMinute:       submitUserPerMin * submitIPMultiplier,
		HintUnlockUserPerMinute: hintUnlockUserPerMin,
		CommentPerMinute:        rateLimitDefault(settings.RateLimitCommentPerMinute, defaultCommentPerMinute),
		RatingPerMinute:         defaultRatingPerMinute,
	}, nil
}

var rateLimitRedisErrors = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "rate_limit_redis_errors_total",
		Help: "Total number of Redis errors encountered by rate-limit middleware.",
	},
	[]string{"limiter"},
)

// RateLimit returns a Redis-backed rate-limit middleware for the given key prefix and limit.
// Falls back to an in-memory counter if Redis is unavailable; errors are counted in prometheus.
// An empty or errored keyFunc result falls back to the client IP.
func RateLimit(client *redis.Client, keyPrefix string, limit int64, window time.Duration, keyFunc func(r *http.Request) (string, error), log logkit.Logger) func(next http.Handler) http.Handler {
	kf := func(r *http.Request) (string, error) {
		k, err := keyFunc(r)
		if err != nil || k == "" {
			return kitMiddleware.GetClientIPFromContext(r.Context()), nil
		}

		return k, nil
	}

	counter, err := httprateredis.NewRedisLimitCounter(&httprateredis.Config{
		Client:    client,
		PrefixKey: cache.KeyLimiterPrefix + keyPrefix,
		OnError: func(err error) {
			if log != nil {
				log.WithError(err).WithFields(map[string]any{"key_prefix": keyPrefix}).Warn("middleware - RateLimit: Redis error, using in-memory fallback")
			}

			rateLimitRedisErrors.WithLabelValues(keyPrefix).Inc()
		},
	})
	if err != nil {
		log.WithError(err).Fatal("failed to create rate limit store")

		return nil
	}

	return httprate.Limit(int(limit), window,
		httprate.WithKeyFuncs(kf),
		httprate.WithLimitCounter(counter),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrTooManyRequests))
		}),
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
	counter, err := httprateredis.NewRedisLimitCounter(&httprateredis.Config{
		Client:    client,
		PrefixKey: DynamicLimitKeyPrefix + keyPrefix,
		OnError: func(err error) {
			log.WithError(err).WithFields(map[string]any{"key_prefix": keyPrefix}).Warn("dynamic rate limit: Redis error, using in-memory fallback")
			rateLimitRedisErrors.WithLabelValues(keyPrefix).Inc()
		},
	})
	if err != nil {
		log.WithError(err).Fatal("failed to create dynamic rate limit store")

		return nil
	}

	kf := func(r *http.Request) (string, error) {
		k, err := keyFunc(r)
		if err != nil || k == "" {
			return kitMiddleware.GetClientIPFromContext(r.Context()), nil
		}

		return k, nil
	}
	limiter := httprate.NewRateLimiter(int(DynamicLimitFallback), window,
		httprate.WithKeyFuncs(kf),
		httprate.WithLimitCounter(counter),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrTooManyRequests))
		}),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			cfg, err := cache.Get(ctx, getter)
			if err != nil {
				log.WithError(err).Warn("rate limit config cache get failed, using stale config")

				cfg = cache.GetStale()
				if cfg == nil {
					cfg = defaultFallbackConfig
				}
			}

			limit := getLimit(cfg)
			if limit <= 0 {
				limit = DynamicLimitFallback
			}

			r = r.WithContext(httprate.WithRequestLimit(ctx, int(limit)))
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
		kf := func(r *http.Request) (string, error) {
			k, err := s.KeyFunc(r)
			if err != nil || k == "" {
				return kitMiddleware.GetClientIPFromContext(r.Context()), nil
			}

			return k, nil
		}

		counter, err := httprateredis.NewRedisLimitCounter(&httprateredis.Config{
			Client:    client,
			PrefixKey: cache.KeyLimiterPrefix + s.KeyPrefix,
			OnError: func(err error) {
				log.WithError(err).WithFields(logkit.Fields{"key_prefix": s.KeyPrefix}).Warn("middleware - CombinedRateLimit: Redis error, using in-memory fallback")
				rateLimitRedisErrors.WithLabelValues(s.KeyPrefix).Inc()
			},
		})
		if err != nil {
			log.WithError(err).WithFields(logkit.Fields{"key_prefix": s.KeyPrefix}).Fatal("middleware - CombinedRateLimit: failed to create store")

			return nil
		}

		mw := httprate.Limit(int(s.Limit), s.Window,
			httprate.WithKeyFuncs(kf),
			httprate.WithLimitCounter(counter),
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrTooManyRequests))
			}),
		)
		mws = append(mws, mw)
	}

	return func(next http.Handler) http.Handler {
		h := next

		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}

		return h
	}
}

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
	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody).WithContext(ctx)
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

const maxConcurrentRatelimitAudit = 64

var ratelimitAuditSem = semaphore.NewWeighted(maxConcurrentRatelimitAudit)

// SubmitRateLimitWithAudit returns a middleware that enforces dynamic per-IP and per-user rate limits
// on flag submission. When a limit is hit, a rate-limited submission record is saved asynchronously
// via saveRatelimitedSubmissionAsync for audit purposes before returning 429.
func SubmitRateLimitWithAudit(
	client *redis.Client,
	ipKeyPrefix, userKeyPrefix string,
	window time.Duration,
	cache *RateLimitConfigCache,
	getter SettingsGetter,
	ipKeyFunc, userIDKeyFunc func(*http.Request) (string, error),
	log logkit.Logger,
	submissionUC usecase.SubmissionUseCase,
	ratelimitAuditWG *sync.WaitGroup,
) func(next http.Handler) http.Handler {
	ipCounter, err := httprateredis.NewRedisLimitCounter(&httprateredis.Config{
		Client:    client,
		PrefixKey: DynamicLimitKeyPrefix + ipKeyPrefix,
		OnError: func(err error) {
			log.WithError(err).WithFields(map[string]any{"limiter": ipKeyPrefix}).Warn("submit rate limit: Redis error (IP), using in-memory fallback")
			rateLimitRedisErrors.WithLabelValues(ipKeyPrefix).Inc()
		},
	})
	if err != nil {
		log.WithError(err).Fatal("failed to create submit rate limit store (IP)")

		return nil
	}

	userCounter, err := httprateredis.NewRedisLimitCounter(&httprateredis.Config{
		Client:    client,
		PrefixKey: DynamicLimitKeyPrefix + userKeyPrefix,
		OnError: func(err error) {
			log.WithError(err).WithFields(map[string]any{"limiter": userKeyPrefix}).Warn("submit rate limit: Redis error (user), using in-memory fallback")
			rateLimitRedisErrors.WithLabelValues(userKeyPrefix).Inc()
		},
	})
	if err != nil {
		log.WithError(err).Fatal("failed to create submit rate limit store (user)")

		return nil
	}

	onRateLimited := func(r *http.Request) {
		saveRatelimitedSubmissionAsync(r, submissionUC, log, ratelimitAuditWG)
	}
	ipKeyFn := func(r *http.Request) (string, error) {
		k, err := ipKeyFunc(r)
		if err != nil || k == "" {
			return kitMiddleware.GetClientIPFromContext(r.Context()), nil
		}

		return k, nil
	}
	userKeyFn := func(r *http.Request) (string, error) {
		k, err := userIDKeyFunc(r)
		if err != nil || k == "" {
			return kitMiddleware.GetClientIPFromContext(r.Context()), nil
		}

		return k, nil
	}
	ipLimiter := httprate.NewRateLimiter(int(DynamicLimitFallback), window,
		httprate.WithKeyFuncs(ipKeyFn),
		httprate.WithLimitCounter(ipCounter),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			onRateLimited(r)
			httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrTooManyRequests))
		}),
	)
	userLimiter := httprate.NewRateLimiter(int(DynamicLimitFallback), window,
		httprate.WithKeyFuncs(userKeyFn),
		httprate.WithLimitCounter(userCounter),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			onRateLimited(r)
			httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrTooManyRequests))
		}),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			cfg, err := cache.Get(ctx, getter)
			if err != nil {
				log.WithError(err).Warn("rate limit config cache get failed, using stale config")

				cfg = cache.GetStale()
				if cfg == nil {
					cfg = defaultFallbackConfig
				}
			}

			ipLimit := int64(cfg.SubmitIPPerMinute)
			if ipLimit <= 0 {
				ipLimit = DynamicLimitFallback
			}

			userLimit := int64(cfg.SubmitUserPerMinute)
			if userLimit <= 0 {
				userLimit = DynamicLimitFallback
			}

			r = r.WithContext(httprate.WithRequestLimit(ctx, int(ipLimit)))
			ipLimiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r = r.WithContext(httprate.WithRequestLimit(r.Context(), int(userLimit)))
				userLimiter.Handler(next).ServeHTTP(w, r)
			})).ServeHTTP(w, r)
		})
	}
}

// saveRatelimitedSubmissionAsync records a rate-limited flag submission in the background.
// Uses a semaphore (max 64) to bound concurrency. Skips silently when the semaphore is full.
func saveRatelimitedSubmissionAsync(r *http.Request, submissionUC usecase.SubmissionUseCase, log logkit.Logger, wg *sync.WaitGroup) {
	if submissionUC == nil {
		return
	}

	if !ratelimitAuditSem.TryAcquire(1) {
		log.Warn("submit rate limit audit: semaphore full, dropping ratelimited submission record")

		return
	}

	user, ok := GetUser(r.Context())
	if !ok || user == nil || user.TeamID == nil {
		ratelimitAuditSem.Release(1)

		return
	}

	challengeIDStr := chi.URLParam(r, "challengeID")

	challengeID, err := uuid.Parse(challengeIDStr)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{"challengeID": challengeIDStr}).Warn("submit rate limit audit: invalid challengeID, skipping audit record")
		ratelimitAuditSem.Release(1)

		return
	}

	userID := user.ID
	teamID := *user.TeamID
	ip := kitMiddleware.GetClientIPFromContext(r.Context())
	body := func() {
		defer ratelimitAuditSem.Release(1)

		logCtx, cancel := context.WithTimeout(context.Background(), ratelimitAuditTimeout)
		defer cancel()

		logErr := submissionUC.LogRateLimited(logCtx, userID, teamID, challengeID, ip)
		if logErr != nil {
			log.WithError(logErr).Warn("submit rate limit audit: LogRateLimited failed")
		}
	}

	if wg != nil {
		wg.Go(body)
	} else {
		go body()
	}
}
