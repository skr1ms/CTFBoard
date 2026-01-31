package middleware

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

func RequireTeam(competitionMode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok {
				httputil.RenderError(w, r, http.StatusUnauthorized, "unauthorized")
				return
			}

			if user.Role == entity.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			if user.TeamID == nil {
				httputil.RenderErrorWithCode(w, r, http.StatusForbidden, entityError.ErrNoTeamSelected.Error(), "no_team_selected")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
