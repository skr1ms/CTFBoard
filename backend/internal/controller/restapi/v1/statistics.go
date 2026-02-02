package v1

import (
	"net/http"
	"strconv"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

const (
	defaultScoreboardHistoryLimit = 10
	maxScoreboardHistoryLimit     = 100
)

// Get general statistics
// (GET /statistics/general)
func (h *Server) GetStatisticsGeneral(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsUC.GetGeneralStats(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetStatisticsGeneral")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromGeneralStats(stats))
}

// Get challenge statistics
// (GET /statistics/challenges)
func (h *Server) GetStatisticsChallenges(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsUC.GetChallengeStats(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetStatisticsChallenges")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromChallengeStatsList(stats))
}

// Get challenge detail statistics
// (GET /statistics/challenges/{id})
func (h *Server) GetStatisticsChallengesId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	stats, err := h.statsUC.GetChallengeDetailStats(r.Context(), id.String())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetStatisticsChallengesId")
		handleError(w, r, err)
		return
	}
	if stats == nil {
		handleError(w, r, entityError.ErrChallengeNotFound)
		return
	}

	RenderOK(w, r, response.FromChallengeDetailStats(stats))
}

// Get scoreboard history
// (GET /statistics/scoreboard)
func (h *Server) GetStatisticsScoreboard(w http.ResponseWriter, r *http.Request) {
	limit := defaultScoreboardHistoryLimit
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
			if limit > maxScoreboardHistoryLimit {
				limit = maxScoreboardHistoryLimit
			}
		}
	}

	stats, err := h.statsUC.GetScoreboardHistory(r.Context(), limit)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetStatisticsScoreboard")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromScoreboardHistoryList(stats))
}

// Get scoreboard graph
// (GET /scoreboard/graph)
func (h *Server) GetScoreboardGraph(w http.ResponseWriter, r *http.Request, params openapi.GetScoreboardGraphParams) {
	topN := defaultScoreboardHistoryLimit
	if params.Top != nil && *params.Top > 0 {
		topN = *params.Top
		if topN > maxScoreboardHistoryLimit {
			topN = maxScoreboardHistoryLimit
		}
	}

	graph, err := h.statsUC.GetScoreboardGraph(r.Context(), topN)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetScoreboardGraph")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromScoreboardGraph(graph))
}
