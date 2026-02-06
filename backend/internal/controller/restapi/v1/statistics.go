package v1

import (
	"net/http"
	"strconv"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
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
	stats, err := h.comp.StatsUC.GetGeneralStats(r.Context())
	if h.OnError(w, r, err, "GetStatisticsGeneral", "GetGeneralStats") {
		return
	}

	helper.RenderOK(w, r, response.FromGeneralStats(stats))
}

// Get challenge statistics
// (GET /statistics/challenges)
func (h *Server) GetStatisticsChallenges(w http.ResponseWriter, r *http.Request) {
	stats, err := h.comp.StatsUC.GetChallengeStats(r.Context())
	if h.OnError(w, r, err, "GetStatisticsChallenges", "GetChallengeStats") {
		return
	}

	helper.RenderOK(w, r, response.FromChallengeStatsList(stats))
}

// Get challenge detail statistics
// (GET /statistics/challenges/{id})
func (h *Server) GetStatisticsChallengesId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	stats, err := h.comp.StatsUC.GetChallengeDetailStats(r.Context(), id.String())
	if err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - GetStatisticsChallengesId")
		helper.HandleError(w, r, err)
		return
	}
	if stats == nil {
		helper.HandleError(w, r, entityError.ErrChallengeNotFound)
		return
	}

	helper.RenderOK(w, r, response.FromChallengeDetailStats(stats))
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

	stats, err := h.comp.StatsUC.GetScoreboardHistory(r.Context(), limit)
	if h.OnError(w, r, err, "GetStatisticsScoreboard", "GetScoreboardHistory") {
		return
	}

	helper.RenderOK(w, r, response.FromScoreboardHistoryList(stats))
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

	graph, err := h.comp.StatsUC.GetScoreboardGraph(r.Context(), topN)
	if h.OnError(w, r, err, "GetScoreboardGraph", "GetScoreboardGraph") {
		return
	}

	helper.RenderOK(w, r, response.FromScoreboardGraph(graph))
}
