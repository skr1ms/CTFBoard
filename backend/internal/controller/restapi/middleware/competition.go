package middleware

import (
	"net/http"
	"time"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// CompetitionActive returns a middleware that rejects requests when submission is not allowed
// (competition not started, ended, or paused), returning a status-specific error.
func CompetitionActive(competitionUC usecase.CompetitionUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				httputil.HandleError(w, r, errmap.MapAppError(err))

				return
			}

			now := time.Now()
			if !comp.IsSubmissionAllowedAt(now) {
				var appErr error

				switch comp.GetStatusAt(now) { //nolint:exhaustive // Active/Frozen allow submission and never reach this branch
				case domain.CompetitionStatusNotStarted:
					appErr = apperr.ErrCompetitionNotStarted
				case domain.CompetitionStatusEnded:
					appErr = apperr.ErrCompetitionEnded
				case domain.CompetitionStatusPaused:
					appErr = apperr.ErrCompetitionPaused
				default:
					appErr = apperr.ErrSubmissionNotAllowed
				}

				httputil.HandleError(w, r, errmap.MapAppError(appErr))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CompetitionEnded returns a middleware that rejects requests when the competition has not yet effectively ended.
// Used for endpoints that are only available post-competition (e.g. comments).
func CompetitionEnded(competitionUC usecase.CompetitionUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				httputil.HandleError(w, r, errmap.MapAppError(err))

				return
			}

			now := time.Now()
			if !comp.IsEffectivelyEnded(now) {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrCommentsAvailableAfterEnd))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
