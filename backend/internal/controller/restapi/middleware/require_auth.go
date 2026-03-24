package middleware

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func RequireAuth(check func(*domain.User) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok || user == nil {
				httputil.HandleError(w, r, httperr.ErrNotAuthenticated())
				return
			}
			if user.Role == domain.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}
			if err := check(user); err != nil {
				httputil.HandleError(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
