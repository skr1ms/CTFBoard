package middleware

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
)

const setupCheckTTL = 5 * time.Second

// SetupStatusChecker is satisfied by SetupUseCase.
type SetupStatusChecker interface {
	IsComplete(ctx context.Context) (bool, error)
}

type SetupAllowlist struct {
	Exact    []string
	Prefixes []string
}

type setupCache struct {
	mu        sync.RWMutex
	complete  bool
	expiresAt time.Time
}

func (c *setupCache) get() (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Now().Before(c.expiresAt) {
		return c.complete, true
	}

	return false, false
}

func (c *setupCache) set(complete bool) {
	if !complete {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.complete = complete
	c.expiresAt = time.Now().Add(setupCheckTTL)
}

// SetupRequired returns middleware that blocks all requests with 503 when the
// platform setup has not been completed, unless the path is in the allowlist.
//
// Completed setup status is cached for setupCheckTTL (5 s) to avoid a Redis/DB
// round-trip on every post-setup request. Incomplete status is intentionally not
// cached so the first request after a successful setup cannot be blocked by a
// stale negative entry.
func SetupRequired(uc SetupStatusChecker, allowlist SetupAllowlist) func(http.Handler) http.Handler {
	cache := &setupCache{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if allowlist.Allows(r.URL.Path) {
				next.ServeHTTP(w, r)

				return
			}

			// Use cached value when fresh.
			if complete, ok := cache.get(); ok {
				if complete {
					next.ServeHTTP(w, r)

					return
				}

				writeSetupRequired(w, r)

				return
			}

			// Cache miss - query the use case.
			complete, err := uc.IsComplete(r.Context())
			if err != nil {
				httputil.HandleError(w, r, errmap.MapAppError(err))

				return
			}

			cache.set(complete)

			if complete {
				next.ServeHTTP(w, r)

				return
			}

			writeSetupRequired(w, r)
		})
	}
}

func (a SetupAllowlist) Allows(path string) bool {
	if slices.Contains(a.Exact, path) {
		return true
	}

	for _, prefix := range a.Prefixes {
		if len(prefix) > 0 && len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

func writeSetupRequired(w http.ResponseWriter, r *http.Request) {
	httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrSetupRequired))
}
