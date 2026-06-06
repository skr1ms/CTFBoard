package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/semaphore"
)

const (
	ratelimitAuditTimeout       = 5 * time.Second
	maxConcurrentRatelimitAudit = 64
)

var ratelimitAuditSem = semaphore.NewWeighted(maxConcurrentRatelimitAudit)

// RateLimitedSubmissionLogger is the minimal interface used to audit rejected flag submissions.
type RateLimitedSubmissionLogger interface {
	LogRateLimited(ctx context.Context, userID, teamID, challengeID uuid.UUID, ip string) error
}

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
	submissionUC RateLimitedSubmissionLogger,
	ratelimitAuditWG *sync.WaitGroup,
) func(next http.Handler) http.Handler {
	ipCounter, err := newRedisLimitCounter(
		client,
		DynamicLimitKeyPrefix+ipKeyPrefix,
		ipKeyPrefix,
		"submit rate limit: Redis error (IP), using in-memory fallback",
		log,
	)
	if err != nil {
		log.WithError(err).Fatal("failed to create submit rate limit store (IP)")

		return nil
	}

	userCounter, err := newRedisLimitCounter(
		client,
		DynamicLimitKeyPrefix+userKeyPrefix,
		userKeyPrefix,
		"submit rate limit: Redis error (user), using in-memory fallback",
		log,
	)
	if err != nil {
		log.WithError(err).Fatal("failed to create submit rate limit store (user)")

		return nil
	}

	onRateLimited := func(r *http.Request) {
		saveRatelimitedSubmissionAsync(r, submissionUC, log, ratelimitAuditWG)
	}
	ipLimiter := httprate.NewRateLimiter(int(DynamicLimitFallback), window,
		httprate.WithKeyFuncs(fallbackKeyFunc(ipKeyFunc)),
		httprate.WithLimitCounter(ipCounter),
		httprate.WithLimitHandler(tooManyRequestsHandler(onRateLimited)),
	)
	userLimiter := httprate.NewRateLimiter(int(DynamicLimitFallback), window,
		httprate.WithKeyFuncs(fallbackKeyFunc(userIDKeyFunc)),
		httprate.WithLimitCounter(userCounter),
		httprate.WithLimitHandler(tooManyRequestsHandler(onRateLimited)),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := rateLimitConfigOrFallback(log, r, cache, getter)

			ipLimit := int64(cfg.SubmitIPPerMinute)
			if ipLimit <= 0 {
				ipLimit = DynamicLimitFallback
			}

			userLimit := int64(cfg.SubmitUserPerMinute)
			if userLimit <= 0 {
				userLimit = DynamicLimitFallback
			}

			r = r.WithContext(httprate.WithRequestLimit(r.Context(), int(ipLimit)))
			ipLimiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r = r.WithContext(httprate.WithRequestLimit(r.Context(), int(userLimit)))
				userLimiter.Handler(next).ServeHTTP(w, r)
			})).ServeHTTP(w, r)
		})
	}
}

// saveRatelimitedSubmissionAsync records a rate-limited flag submission in the background.
// Uses a semaphore (max 64) to bound concurrency. Skips silently when the semaphore is full.
func saveRatelimitedSubmissionAsync(r *http.Request, submissionUC RateLimitedSubmissionLogger, log logkit.Logger, wg *sync.WaitGroup) {
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
	auditCtx := context.WithoutCancel(r.Context())
	body := func(ctx context.Context) {
		defer ratelimitAuditSem.Release(1)

		logCtx, cancel := context.WithTimeout(ctx, ratelimitAuditTimeout)
		defer cancel()

		logErr := submissionUC.LogRateLimited(logCtx, userID, teamID, challengeID, ip)
		if logErr != nil {
			log.WithError(logErr).Warn("submit rate limit audit: LogRateLimited failed")
		}
	}

	if wg != nil {
		wg.Go(func() { body(auditCtx) })
	} else {
		go body(auditCtx)
	}
}
