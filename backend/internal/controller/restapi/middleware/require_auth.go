package middleware

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// RequireAuth returns a middleware that rejects unauthenticated requests and applies an optional
// per-user check. Admin users bypass the check function and are always allowed through.
func RequireAuth(check func(*domain.User) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok || user == nil {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrNotAuthenticated))

				return
			}

			if isAdmin(r.Context()) {
				next.ServeHTTP(w, r)

				return
			}

			err := check(user)
			if err != nil {
				httputil.HandleError(w, r, errmap.MapAppError(err))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
