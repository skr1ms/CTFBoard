package middleware

import (
	"context"
	"net/http"
	"strings"
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
	c.mu.Lock()
	defer c.mu.Unlock()

	c.complete = complete
	c.expiresAt = time.Now().Add(setupCheckTTL)
}

// SetupRequired returns middleware that blocks all requests with 503 when the
// platform setup has not been completed, unless the path is in the allowlist.
//
// The setup status is cached for setupCheckTTL (5 s) to avoid a Redis/DB round-trip
// on every request. Once setup is complete the cache is never invalidated - setup can
// only transition from false -> true, never the other way.
func SetupRequired(uc SetupStatusChecker, allowedPrefixes []string) func(http.Handler) http.Handler {
	cache := &setupCache{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAllowed(r.URL.Path, allowedPrefixes) {
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
				// On error, allow the request through - don't block the platform for transient failures.
				next.ServeHTTP(w, r)

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

func isAllowed(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

func writeSetupRequired(w http.ResponseWriter, r *http.Request) {
	httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrSetupRequired))
}
