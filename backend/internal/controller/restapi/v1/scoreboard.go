package v1

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
)

// Get scoreboard
// (GET /scoreboard)
func (h *Server) GetScoreboard(w http.ResponseWriter, r *http.Request) {
	entries, err := h.solveUC.GetScoreboard(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetScoreboard")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromScoreboardList(entries))
}

// Get first blood
// (GET /challenges/{ID}/first-blood)
func (h *Server) GetChallengesIDFirstBlood(w http.ResponseWriter, r *http.Request, ID string) {
	challengeuuid, err := uuid.Parse(ID)
	if err != nil {
		RenderInvalidID(w, r)
		return
	}

	entry, err := h.solveUC.GetFirstBlood(r.Context(), challengeuuid)
	if err != nil {
		if errors.Is(err, entityError.ErrSolveNotFound) {
			RenderError(w, r, http.StatusNotFound, "no solves yet")
			return
		}
		h.logger.WithError(err).Error("restapi - v1 - GetChallengesIDFirstBlood")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromFirstBlood(entry))
}
