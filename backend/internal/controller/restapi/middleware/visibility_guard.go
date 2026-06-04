package middleware

import (
	"context"
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
)

// VisibilityConfigGetter is the minimal interface for reading a visibility config key.
type VisibilityConfigGetter interface {
	GetString(ctx context.Context, key, defaultVal string) string
}

// VisibilityGuard returns middleware that enforces a visibility config key read from
// the competition parameter store. The getter owns caching/invalidation, so this
// middleware reads through it on every request and does not add a stale TTL layer.
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
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAdmin(r.Context()) {
				next.ServeHTTP(w, r)

				return
			}

			visibility := getter.GetString(r.Context(), configKey, "public")
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
