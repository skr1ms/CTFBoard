package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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

func (rs *statisticsRoutes) GetGeneralStats(w http.ResponseWriter, r *http.Request) {
	stats, err := rs.statsUC.GetGeneralStats(r.Context())
	if err != nil {
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get general stats")
		return
	}

	render.JSON(w, r, stats)
}

func (rs *statisticsRoutes) GetChallengeStats(w http.ResponseWriter, r *http.Request) {
	stats, err := rs.statsUC.GetChallengeStats(r.Context())
	if err != nil {
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get challenge stats")
		return
	}

	render.JSON(w, r, stats)
}

func (rs *statisticsRoutes) GetScoreboardHistory(w http.ResponseWriter, r *http.Request) {
	limit := 10

	stats, err := rs.statsUC.GetScoreboardHistory(r.Context(), limit)
	if err != nil {
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get scoreboard history")
		return
	}

	render.JSON(w, r, stats)
}
