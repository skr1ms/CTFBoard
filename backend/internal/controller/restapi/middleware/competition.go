package middleware

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

func CompetitionActive(competitionUC *competition.CompetitionUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get competition status")
				return
			}

			if !comp.IsSubmissionAllowed() {
				status := comp.GetStatus()
				var msg string
				switch status {
				case entity.CompetitionStatusNotStarted:
					msg = "competition has not started yet"
				case entity.CompetitionStatusEnded:
					msg = "competition has ended"
				case entity.CompetitionStatusPaused:
					msg = "competition is paused"
				case entity.CompetitionStatusActive, entity.CompetitionStatusFrozen:
					msg = "submissions are not allowed at this time"
				default:
					msg = "submissions are not allowed at this time"
				}
				httputil.RenderError(w, r, http.StatusForbidden, msg)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CompetitionEnded(competitionUC *competition.CompetitionUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get competition status")
				return
			}
			if comp.GetStatus() != entity.CompetitionStatusEnded {
				httputil.RenderError(w, r, http.StatusForbidden, "comments available only after competition has ended")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
