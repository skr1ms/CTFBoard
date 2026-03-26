package middleware

import (
	"net/http"
	"time"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func CompetitionActive(competitionUC usecase.CompetitionUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				httputil.HandleError(w, r, err)

				return
			}

			now := time.Now()
			if !comp.IsSubmissionAllowedAt(now) {
				var httpErr *httperr.HTTPError

				switch comp.GetStatusAt(now) { //nolint:exhaustive // Active/Frozen allow submission and never reach this branch
				case domain.CompetitionStatusNotStarted:
					httpErr = httperr.ErrCompetitionNotStarted
				case domain.CompetitionStatusEnded:
					httpErr = httperr.ErrCompetitionEnded
				case domain.CompetitionStatusPaused:
					httpErr = httperr.ErrCompetitionPaused
				default:
					httpErr = httperr.ErrSubmissionNotAllowed
				}

				httputil.HandleError(w, r, httpErr)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CompetitionEnded(competitionUC usecase.CompetitionUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				httputil.HandleError(w, r, err)

				return
			}

			now := time.Now()
			if !comp.IsEffectivelyEnded(now) {
				httputil.HandleError(w, r, httperr.ErrCommentsAvailableAfterEnd)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
