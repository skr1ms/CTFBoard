package middleware

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
)

func RequireVerified(verifyEmails bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !verifyEmails {
				next.ServeHTTP(w, r)
				return
			}

			user, ok := GetUser(r.Context())
			if !ok || user == nil {
				httputil.HandleError(w, r, httperr.ErrNotAuthenticated)
				return
			}

			if user.Role == entity.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			if !user.IsVerified {
				httputil.HandleError(w, r, httperr.ErrEmailNotVerified)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
