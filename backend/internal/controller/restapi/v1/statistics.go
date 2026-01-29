package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
)

type statisticsRoutes struct {
	statsUC *usecase.StatisticsUseCase
	logger  logger.Logger
}

func NewStatisticsRoutes(router chi.Router, statsUC *usecase.StatisticsUseCase, logger logger.Logger, jwtService *jwt.JWTService) {
	routes := statisticsRoutes{
		statsUC: statsUC,
		logger:  logger,
	}

	router.Get("/statistics/general", routes.GetGeneralStats)
	router.Get("/statistics/challenges", routes.GetChallengeStats)
	router.Get("/statistics/scoreboard", routes.GetScoreboardHistory)
}

// @Summary      Get general statistics
// @Description  Returns general platform statistics (counts of users, teams, challenges, solves)
// @Tags         Statistics
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  entity.GeneralStats
// @Failure      500  {object}  ErrorResponse
// @Router       /statistics/general [get]
func (rs *statisticsRoutes) GetGeneralStats(w http.ResponseWriter, r *http.Request) {
	stats, err := rs.statsUC.GetGeneralStats(r.Context())
	if err != nil {
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get general stats")
		return
	}

	httputil.RenderOK(w, r, stats)
}

// @Summary      Get challenge statistics
// @Description  Returns statistics for all challenges including solve counts
// @Tags         Statistics
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   entity.ChallengeStats
// @Failure      500  {object}  ErrorResponse
// @Router       /statistics/challenges [get]
func (rs *statisticsRoutes) GetChallengeStats(w http.ResponseWriter, r *http.Request) {
	stats, err := rs.statsUC.GetChallengeStats(r.Context())
	if err != nil {
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get challenge stats")
		return
	}

	httputil.RenderOK(w, r, stats)
}

// @Summary      Get scoreboard history
// @Description  Returns scoreboard history for graph visualization. Default limit 10 (max 50)
// @Tags         Statistics
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   entity.ScoreboardHistoryEntry
// @Failure      500  {object}  ErrorResponse
// @Router       /statistics/scoreboard [get]
func (rs *statisticsRoutes) GetScoreboardHistory(w http.ResponseWriter, r *http.Request) {
	limit := 10

	stats, err := rs.statsUC.GetScoreboardHistory(r.Context(), limit)
	if err != nil {
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get scoreboard history")
		return
	}

	httputil.RenderOK(w, r, stats)
}
