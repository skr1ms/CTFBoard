package middleware

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

func ChallengeVisibility(competitionUC *competition.CompetitionUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, hasUser := GetUser(r.Context())
			if hasUser && user.Role == entity.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get competition status")
				return
			}

			status := comp.GetStatus()
			if status == entity.CompetitionStatusNotStarted {
				httputil.RenderError(w, r, http.StatusForbidden, "challenges are not yet available")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
