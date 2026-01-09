package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/skr1ms/CTFBoard/internal/controller/http/v1/response"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/logger"
)

type scoreboardRoutes struct {
	solveUC *usecase.SolveUseCase
	logger  logger.Interface
}

func NewScoreboardRoutes(router chi.Router, solveUC *usecase.SolveUseCase, logger logger.Interface) {
	routes := scoreboardRoutes{
		solveUC: solveUC,
		logger:  logger,
	}

	router.Get("/scoreboard", routes.GetScoreboard)
}

// @Summary      Get scoreboard
// @Description  Returns current scoreboard state sorted by points descending
// @Tags         Scoreboard
// @Produce      json
// @Success      200  {array}   response.ScoreboardEntryResponse
// @Router       /scoreboard [get]
func (h *scoreboardRoutes) GetScoreboard(w http.ResponseWriter, r *http.Request) {
	entries, err := h.solveUC.GetScoreboard(r.Context())
	if err != nil {
		h.logger.Error("http - v1 - GetScoreboard - GetScoreboard", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	res := make([]response.ScoreboardEntryResponse, 0)
	for _, entry := range entries {
		res = append(res, response.ScoreboardEntryResponse{
			TeamId:   entry.TeamId,
			TeamName: entry.TeamName,
			Points:   entry.Points,
		})
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}
