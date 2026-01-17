package middleware

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase"
)

func CompetitionActive(competitionUC *usecase.CompetitionUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			comp, err := competitionUC.Get(r.Context())
			if err != nil {
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, map[string]string{"error": "failed to get competition status"})
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
				default:
					msg = "submissions are not allowed at this time"
				}
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, map[string]string{"error": msg})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
