package middleware

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
)

func RequireTeam() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			allowed, err := requireTeamAllowed(user, ok)
			if err != nil {
				httputil.HandleError(w, r, err)
				return
			}
			if !allowed {
				httputil.HandleError(w, r, httperr.ErrTeamModeRequired)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireTeamOrNotFound() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok || user == nil {
				httputil.HandleError(w, r, httperr.ErrNotAuthenticated)
				return
			}
			if user.Role == entity.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}
			if user.TeamID == nil {
				httputil.HandleError(w, r, httperr.ErrUserNotInTeam)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requireTeamAllowed(user *entity.User, hasUser bool) (bool, error) {
	if !hasUser || user == nil {
		return false, httperr.ErrNotAuthenticated
	}
	if user.Role == entity.RoleAdmin {
		return true, nil
	}
	return user.TeamID != nil, nil
}
