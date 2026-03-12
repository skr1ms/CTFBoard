package v1

import (
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
)

const (
	defaultScoreboardHistoryLimit = 10
)

func forceLiveFromParams(r *http.Request, live *bool) bool {
	if live == nil || !*live {
		return false
	}
	user, ok := middleware.GetUser(r.Context())
	return ok && user.Role == entity.RoleAdmin
}

// Get general statistics
// (GET /statistics/general)
func (h *Server) GetStatisticsGeneral(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsGeneralParams) {
	forceLive := forceLiveFromParams(r, params.Live)
	stats, err := h.comp.StatsUC.GetGeneralStats(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsGeneral", "GetGeneralStats") {
		return
	}
	helper.RenderOK(w, r, response.FromGeneralStats(stats))
}

// Get challenge statistics
// (GET /statistics/challenges)
func (h *Server) GetStatisticsChallenges(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsChallengesParams) {
	forceLive := forceLiveFromParams(r, params.Live)
	stats, err := h.comp.StatsUC.GetChallengeStats(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsChallenges", "GetChallengeStats") {
		return
	}
	helper.RenderOK(w, r, response.FromChallengeStatsList(stats))
}

// Get challenge detail statistics
// (GET /statistics/challenges/{ID})
func (h *Server) GetStatisticsChallengesID(w http.ResponseWriter, r *http.Request, id openapi_types.UUID, params openapi.GetStatisticsChallengesIDParams) {
	forceLive := forceLiveFromParams(r, params.Live)
	stats, err := h.comp.StatsUC.GetChallengeDetailStats(r.Context(), id.String(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsChallengesID", "GetChallengeDetailStats") {
		return
	}
	helper.RenderOK(w, r, response.FromChallengeDetailStats(stats))
}

// Get scoreboard history
// (GET /statistics/scoreboard)
func (h *Server) GetStatisticsScoreboard(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsScoreboardParams) {
	forceLive := forceLiveFromParams(r, params.Live)
	limit := helper.ClampLimit(params.Limit, defaultScoreboardHistoryLimit, competition.MaxScoreboardHistoryLimit)
	stats, err := h.comp.StatsUC.GetScoreboardHistory(r.Context(), limit, forceLive)
	if h.OnError(w, r, err, "GetStatisticsScoreboard", "GetScoreboardHistory") {
		return
	}
	helper.RenderOK(w, r, response.FromScoreboardHistoryList(stats))
}

// Get challenge solve percentages
// (GET /statistics/challenges/solves/percentages)
func (h *Server) GetStatisticsChallengesSolvesPercentages(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsChallengesSolvesPercentagesParams) {
	forceLive := forceLiveFromParams(r, params.Live)
	data, err := h.comp.StatsUC.GetChallengeSolvePercentages(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsChallengesSolvesPercentages", "GetChallengeSolvePercentages") {
		return
	}
	helper.RenderOK(w, r, response.FromChallengeSolvePercentages(data))
}

// Get score distribution
// (GET /statistics/scores/distribution)
func (h *Server) GetStatisticsScoresDistribution(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsScoresDistributionParams) {
	forceLive := forceLiveFromParams(r, params.Live)
	data, err := h.comp.StatsUC.GetScoreDistribution(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsScoresDistribution", "GetScoreDistribution") {
		return
	}
	helper.RenderOK(w, r, response.FromScoreDistribution(data))
}

// Get submission time series
// (GET /statistics/submissions)
func (h *Server) GetStatisticsSubmissions(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsSubmissionsParams) {
	forceLive := forceLiveFromParams(r, params.Live)
	data, err := h.comp.StatsUC.GetSubmissionTimeSeries(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsSubmissions", "GetSubmissionTimeSeries") {
		return
	}
	helper.RenderOK(w, r, response.FromSubmissionTimeSeries(data))
}

// Get submission time series by type
// (GET /statistics/submissions/{type})
func (h *Server) GetStatisticsSubmissionsType(w http.ResponseWriter, r *http.Request, pType openapi.GetStatisticsSubmissionsTypeParamsType, params openapi.GetStatisticsSubmissionsTypeParams) {
	if pType != openapi.Correct && pType != openapi.Incorrect {
		h.OnError(w, r, helper.NewValidationErrorf("type must be 'correct' or 'incorrect'"), "GetStatisticsSubmissionsType", "Type")
		return
	}
	forceLive := forceLiveFromParams(r, params.Live)
	isCorrect := pType == openapi.Correct
	data, err := h.comp.StatsUC.GetSubmissionTimeSeriesByType(r.Context(), isCorrect, forceLive)
	if h.OnError(w, r, err, "GetStatisticsSubmissionsType", "GetSubmissionTimeSeriesByType") {
		return
	}
	helper.RenderOK(w, r, response.FromRegistrationTimeSeries(data))
}

// Get team registration statistics
// (GET /statistics/teams)
func (h *Server) GetStatisticsTeams(w http.ResponseWriter, r *http.Request) {
	data, err := h.comp.StatsUC.GetTeamRegistrationTimeSeries(r.Context())
	if h.OnError(w, r, err, "GetStatisticsTeams", "GetTeamRegistrationTimeSeries") {
		return
	}
	helper.RenderOK(w, r, response.FromRegistrationTimeSeries(data))
}

// Get user registration statistics
// (GET /statistics/users)
func (h *Server) GetStatisticsUsers(w http.ResponseWriter, r *http.Request) {
	data, err := h.comp.StatsUC.GetUserRegistrationTimeSeries(r.Context())
	if h.OnError(w, r, err, "GetStatisticsUsers", "GetUserRegistrationTimeSeries") {
		return
	}
	helper.RenderOK(w, r, response.FromRegistrationTimeSeries(data))
}

// Get scoreboard graph
// (GET /scoreboard/graph)
func (h *Server) GetScoreboardGraph(w http.ResponseWriter, r *http.Request, params openapi.GetScoreboardGraphParams) {
	forceLive := forceLiveFromParams(r, params.Live)
	topN := helper.ClampLimit(params.Top, defaultScoreboardHistoryLimit, competition.MaxScoreboardHistoryLimit)
	graph, err := h.comp.StatsUC.GetScoreboardGraph(r.Context(), topN, forceLive)
	if h.OnError(w, r, err, "GetScoreboardGraph", "GetScoreboardGraph") {
		return
	}
	helper.RenderOK(w, r, response.FromScoreboardGraph(graph))
}

// Get solve matrix (admin)
// (GET /admin/statistics/solve-matrix)
func (h *Server) GetAdminStatisticsSolveMatrix(w http.ResponseWriter, r *http.Request, params openapi.GetAdminStatisticsSolveMatrixParams) {
	forceLive := params.Live != nil && *params.Live
	matrix, err := h.comp.StatsUC.GetSolveMatrix(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetAdminStatisticsSolveMatrix", "GetSolveMatrix") {
		return
	}
	helper.RenderOK(w, r, response.FromSolveMatrixList(matrix))
}
