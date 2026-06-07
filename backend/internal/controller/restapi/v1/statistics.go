package v1

import (
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /statistics/general).
func (h *Server) GetStatisticsGeneral(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsGeneralParams) {
	forceLive := forceLiveFromParams(r, params.Live)

	stats, err := h.comp.StatsUC.GetGeneralStats(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsGeneral", "GetGeneralStats") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromGeneralStats(stats))
}

// (GET /statistics/challenges).
func (h *Server) GetStatisticsChallenges(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsChallengesParams) {
	forceLive := forceLiveFromParams(r, params.Live)

	stats, err := h.comp.StatsUC.GetChallengeStats(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsChallenges", "GetChallengeStats") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromChallengeStatsList(stats))
}

// (GET /statistics/challenges/{ID}).
func (h *Server) GetStatisticsChallengesID(w http.ResponseWriter, r *http.Request, id openapi_types.UUID, params openapi.GetStatisticsChallengesIDParams) {
	forceLive := forceLiveFromParams(r, params.Live)

	stats, err := h.comp.StatsUC.GetChallengeDetailStats(r.Context(), id.String(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsChallengesID", "GetChallengeDetailStats") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromChallengeDetailStats(stats))
}

// (GET /statistics/scoreboard).
func (h *Server) GetStatisticsScoreboard(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsScoreboardParams) {
	forceLive := forceLiveFromParams(r, params.Live)
	limit := helper.ResolveScoreboardHistoryLimit(params.Limit)

	stats, err := h.comp.StatsUC.GetScoreboardHistory(r.Context(), limit, forceLive)
	if h.OnError(w, r, err, "GetStatisticsScoreboard", "GetScoreboardHistory") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromScoreboardHistoryList(stats))
}

// (GET /statistics/challenges/solves/percentages).
func (h *Server) GetStatisticsChallengesSolvesPercentages(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsChallengesSolvesPercentagesParams) {
	forceLive := forceLiveFromParams(r, params.Live)

	data, err := h.comp.StatsUC.GetChallengeSolvePercentages(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsChallengesSolvesPercentages", "GetChallengeSolvePercentages") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromChallengeSolvePercentages(data))
}

// (GET /statistics/scores/distribution).
func (h *Server) GetStatisticsScoresDistribution(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsScoresDistributionParams) {
	forceLive := forceLiveFromParams(r, params.Live)

	data, err := h.comp.StatsUC.GetScoreDistribution(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsScoresDistribution", "GetScoreDistribution") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromScoreDistribution(data))
}

// (GET /statistics/submissions).
func (h *Server) GetStatisticsSubmissions(w http.ResponseWriter, r *http.Request, params openapi.GetStatisticsSubmissionsParams) {
	forceLive := forceLiveFromParams(r, params.Live)

	data, err := h.comp.StatsUC.GetSubmissionTimeSeries(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetStatisticsSubmissions", "GetSubmissionTimeSeries") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromSubmissionTimeSeries(data))
}

// (GET /statistics/submissions/{type}).
func (h *Server) GetStatisticsSubmissionsType(w http.ResponseWriter, r *http.Request, pType openapi.GetStatisticsSubmissionsTypeParamsType, params openapi.GetStatisticsSubmissionsTypeParams) {
	isCorrect, err := request.SubmissionTypeIsCorrect(pType)
	if h.OnError(w, r, err, "GetStatisticsSubmissionsType", "Type") {
		return
	}

	forceLive := forceLiveFromParams(r, params.Live)

	data, err := h.comp.StatsUC.GetSubmissionTimeSeriesByType(r.Context(), isCorrect, forceLive)
	if h.OnError(w, r, err, "GetStatisticsSubmissionsType", "GetSubmissionTimeSeriesByType") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromRegistrationTimeSeries(data))
}

// (GET /statistics/teams).
func (h *Server) GetStatisticsTeams(w http.ResponseWriter, r *http.Request) {
	data, err := h.comp.StatsUC.GetTeamRegistrationTimeSeries(r.Context())
	if h.OnError(w, r, err, "GetStatisticsTeams", "GetTeamRegistrationTimeSeries") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromRegistrationTimeSeries(data))
}

// (GET /statistics/users).
func (h *Server) GetStatisticsUsers(w http.ResponseWriter, r *http.Request) {
	data, err := h.comp.StatsUC.GetUserRegistrationTimeSeries(r.Context())
	if h.OnError(w, r, err, "GetStatisticsUsers", "GetUserRegistrationTimeSeries") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromRegistrationTimeSeries(data))
}

// (GET /scoreboard/graph).
func (h *Server) GetScoreboardGraph(w http.ResponseWriter, r *http.Request, params openapi.GetScoreboardGraphParams) {
	forceLive := forceLiveFromParams(r, params.Live)
	topN := helper.ResolveScoreboardHistoryLimit(params.Top)

	graph, err := h.comp.StatsUC.GetScoreboardGraph(r.Context(), topN, forceLive)
	if h.OnError(w, r, err, "GetScoreboardGraph", "GetScoreboardGraph") {
		return
	}

	setPublicCache(w, cacheMicro, true)
	httputil.RenderOK(w, r, response.FromScoreboardGraph(graph))
}

// (GET /admin/statistics/solve-matrix).
func (h *Server) GetAdminStatisticsSolveMatrix(w http.ResponseWriter, r *http.Request, params openapi.GetAdminStatisticsSolveMatrixParams) {
	forceLive := adminForceLive(params.Live)

	matrix, err := h.comp.StatsUC.GetSolveMatrix(r.Context(), forceLive)
	if h.OnError(w, r, err, "GetAdminStatisticsSolveMatrix", "GetSolveMatrix") {
		return
	}

	httputil.RenderOK(w, r, response.FromSolveMatrixList(matrix))
}

// (GET /admin/statistics/funnel).
func (h *Server) GetAdminStatisticsFunnel(w http.ResponseWriter, r *http.Request, params openapi.GetAdminStatisticsFunnelParams) {
	forceLive := adminForceLive(params.Live)
	limit := helper.ResolveScoreboardHistoryLimit(params.Limit)

	funnel, err := h.comp.StatsUC.GetAdminStatisticsFunnel(r.Context(), limit, forceLive)
	if h.OnError(w, r, err, "GetAdminStatisticsFunnel", "GetAdminStatisticsFunnel") {
		return
	}

	httputil.RenderOK(w, r, response.FromAdminStatisticsFunnel(funnel))
}
