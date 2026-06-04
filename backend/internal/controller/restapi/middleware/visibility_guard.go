package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
)

const visibilityGuardTTL = 30 * time.Second

// VisibilityConfigGetter is the minimal interface for reading a visibility config key.
type VisibilityConfigGetter interface {
	GetString(ctx context.Context, key, defaultVal string) string
}

// VisibilityGuard returns middleware that enforces a visibility config key read from
// the competition parameter store with a 30 s CachedValue TTL.
//
// Value semantics:
//
//	"public"         -> always pass through
//	"private"        -> require authenticated user (401 for guests)
//	"hidden"/"admins" -> 404 for non-admins (prevents endpoint existence leak)
//	anything else    -> 404 for non-admins (fail-closed)
//
// Admin users always bypass the check regardless of the configured value.
func VisibilityGuard(getter VisibilityConfigGetter, configKey string) func(http.Handler) http.Handler {
	cv := cachekit.NewCachedValue[string](context.Background(), configKey, visibilityGuardTTL)

	load := func(ctx context.Context) (string, error) {
		return getter.GetString(ctx, configKey, "public"), nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAdmin(r.Context()) {
				next.ServeHTTP(w, r)

				return
			}

			visibility, err := cv.Get(r.Context(), load)
			if err != nil {
				httputil.HandleError(w, r, errmap.MapAppError(err))

				return
			}

			switch visibility {
			case "public":
				next.ServeHTTP(w, r)
			case "private":
				user, ok := GetUser(r.Context())
				if !ok || user == nil {
					httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrNotAuthenticated))

					return
				}

				next.ServeHTTP(w, r)
			case "hidden", "admins", "admins_only":
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrVisibilityForbidden))

				return
			default:
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrVisibilityForbidden))

				return
			}
		})
	}
}
