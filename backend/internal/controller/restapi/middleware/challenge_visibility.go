package middleware

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// ChallengeVisibility returns a middleware that blocks access to challenges before the competition starts.
// Admin users bypass the check and always have access.
func ChallengeVisibility(competitionUC CompetitionGetter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAdmin(r.Context()) {
				next.ServeHTTP(w, r)

				return
			}

			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				httputil.HandleError(w, r, errmap.MapAppError(err))

				return
			}

			status := comp.GetStatus()
			if status == domain.CompetitionStatusNotStarted {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrCompetitionNotStarted))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
