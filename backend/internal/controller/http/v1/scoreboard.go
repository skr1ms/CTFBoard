package v1

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/skr1ms/CTFBoard/internal/controller/http/v1/response"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/logger"
)

type scoreboardRoutes struct {
	solveUC *usecase.SolveUseCase
	logger  logger.Interface
}

func NewScoreboardRoutes(router chi.Router,
	solveUC *usecase.SolveUseCase,
	logger logger.Interface,
) {
	routes := scoreboardRoutes{
		solveUC: solveUC,
		logger:  logger,
	}

	router.Get("/scoreboard", routes.GetScoreboard)
	router.Get("/challenges/{id}/first-blood", routes.GetFirstBlood)
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

// @Summary      Get first blood
// @Description  Returns the first solver of a challenge
// @Tags         Challenges
// @Produce      json
// @Param        id   path      string  true  "Challenge ID"
// @Success      200  {object}  response.FirstBloodResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /challenges/{id}/first-blood [get]
func (h *scoreboardRoutes) GetFirstBlood(w http.ResponseWriter, r *http.Request) {
	challengeId := chi.URLParam(r, "id")
	if challengeId == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, nil)
		return
	}

	entry, err := h.solveUC.GetFirstBlood(r.Context(), challengeId)
	if err != nil {
		if errors.Is(err, entityError.ErrSolveNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "no solves yet"})
			return
		}
		h.logger.Error("http - v1 - GetFirstBlood - GetFirstBlood", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	res := response.FirstBloodResponse{
		UserId:   entry.UserId,
		Username: entry.Username,
		TeamId:   entry.TeamId,
		TeamName: entry.TeamName,
		SolvedAt: entry.SolvedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}
