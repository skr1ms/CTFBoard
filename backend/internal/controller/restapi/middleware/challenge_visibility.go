package middleware

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
)

func ChallengeVisibility(competitionUC usecase.CompetitionUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if ok && user != nil && user.Role == entity.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				httputil.HandleError(w, r, err)
				return
			}

			status := comp.GetStatus()
			if status == entity.CompetitionStatusNotStarted {
				httputil.HandleError(w, r, httperr.ErrCompetitionNotStarted)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
