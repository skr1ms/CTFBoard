package v1

import (
	"net/http"
	"strconv"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
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
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get general stats")
		return
	}

	res := openapi.EntityGeneralStats{
		UserCount:      ptr(stats.UserCount),
		TeamCount:      ptr(stats.TeamCount),
		ChallengeCount: ptr(stats.ChallengeCount),
		SolveCount:     ptr(stats.SolveCount),
	}

	httputil.RenderOK(w, r, res)
}

// Get challenge statistics
// (GET /statistics/challenges)
func (h *Server) GetStatisticsChallenges(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsUC.GetChallengeStats(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetStatisticsChallenges")
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get challenge stats")
		return
	}

	res := make([]openapi.EntityChallengeStats, len(stats))
	for i, s := range stats {
		res[i] = openapi.EntityChallengeStats{
			ID:         ptr(s.ID.String()),
			Title:      ptr(s.Title),
			Points:     ptr(s.Points),
			SolveCount: ptr(s.SolveCount),
			Category:   ptr(s.Category),
		}
	}

	httputil.RenderOK(w, r, res)
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
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get scoreboard history")
		return
	}

	res := make([]openapi.EntityScoreboardHistoryEntry, len(stats))
	for i, s := range stats {
		res[i] = openapi.EntityScoreboardHistoryEntry{
			TeamID:    ptr(s.TeamID.String()),
			TeamName:  ptr(s.TeamName),
			Points:    ptr(s.Points),
			Timestamp: ptr(s.Timestamp.String()),
		}
	}

	httputil.RenderOK(w, r, res)
}
