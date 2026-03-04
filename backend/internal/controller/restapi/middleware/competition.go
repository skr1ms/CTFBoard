package middleware

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
)

func CompetitionActive(competitionUC usecase.CompetitionUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				httputil.HandleError(w, r, err)
				return
			}

			if !comp.IsSubmissionAllowed() {
				var httpErr *httperr.HTTPError
				switch comp.GetStatus() { //nolint:exhaustive // Active/Frozen allow submission and never reach this branch
				case entity.CompetitionStatusNotStarted:
					httpErr = httperr.ErrCompetitionNotStarted
				case entity.CompetitionStatusEnded:
					httpErr = httperr.ErrCompetitionEnded
				case entity.CompetitionStatusPaused:
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
			if comp.GetStatus() != entity.CompetitionStatusEnded {
				httputil.HandleError(w, r, httperr.ErrCommentsAvailableAfterEnd)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
