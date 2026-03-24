package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const asyncLogTimeout = 5 * time.Second

// Get challenges list
// (GET /challenges)
func (h *Server) GetChallenges(w http.ResponseWriter, r *http.Request, params openapi.GetChallengesParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	var tagID *uuid.UUID
	if params.Tag != nil && *params.Tag != "" {
		id, ok := httputil.ParseUUIDField(w, r, *params.Tag, "tag")
		if !ok {
			return
		}
		tagID = &id
	}

	challenges, err := h.challenge.ChallengeUC.GetAll(r.Context(), user.TeamID, tagID)
	if h.OnError(w, r, err, "GetChallenges", "GetAll") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallengeList(challenges))
}

// Submit flag
// (POST /challenges/{challengeID}/submit)
func (h *Server) PostChallengesChallengeIDSubmit(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.SubmitFlagRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}
	if err := request.ValidateSubmitFlagRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PostChallengesChallengeIDSubmit", "Validate") {
		return
	}
	flag, err := request.SubmitFlagRequestToParams(&req)
	if h.OnError(w, r, err, "PostChallengesChallengeIDSubmit", "RequestConversion") {
		return
	}
	clientIP := kitMiddleware.GetClientIPFromContext(r.Context())
	valid, submitErr := h.challenge.ChallengeUC.SubmitFlag(r.Context(), challengeIDParsed, flag, user.ID, user.TeamID, clientIP)

	if h.OnError(w, r, submitErr, "PostChallengesChallengeIDSubmit", "SubmitFlag") {
		return
	}

	if !valid {
		httputil.RenderOK(w, r, response.FromSubmitFlag(false, "incorrect flag"))
		return
	}

	httputil.RenderOK(w, r, response.FromSubmitFlag(true, "flag accepted"))
}

// Create challenge
// (POST /admin/challenges)
func (h *Server) PostAdminChallenges(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.CreateChallengeRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.CreateChallengeRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminChallenges", "RequestConversion") {
		return
	}
	challenge, err := h.challenge.ChallengeUC.Create(
		r.Context(),
		params.Title, params.Description, params.Category, params.Points, params.InitialValue, params.MinValue, params.Decay, params.Flag,
		params.ConnectionInfo, params.MaxAttempts, params.Position, params.State, params.IsRegex, params.IsCaseInsensitive, params.FlagFormatRegex, params.TagIDs,
	)
	if h.OnError(w, r, err, "PostAdminChallenges", "Create") {
		return
	}

	httputil.RenderCreated(w, r, response.FromChallenge(challenge))
}

// Delete challenge
// (DELETE /admin/challenges/{ID})
func (h *Server) DeleteAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	clientIP := kitMiddleware.GetClientIPFromContext(r.Context())

	err := h.challenge.ChallengeUC.Delete(r.Context(), challengeIDParsed, user.ID, clientIP)
	if h.OnError(w, r, err, "DeleteAdminChallengesID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// Update challenge
// (PUT /admin/challenges/{ID})
func (h *Server) PutAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateChallengeRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.UpdateChallengeRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminChallengesID", "RequestConversion") {
		return
	}
	challenge, err := h.challenge.ChallengeUC.Update(
		r.Context(),
		challengeIDParsed,
		params.Title, params.Description, params.Category, params.Points, params.InitialValue, params.MinValue, params.Decay, params.Flag,
		params.ConnectionInfo, params.MaxAttempts, params.Position, params.State, params.IsRegex, params.IsCaseInsensitive, params.FlagFormatRegex, params.TagIDs,
	)
	if h.OnError(w, r, err, "PutAdminChallengesID", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallenge(challenge))
}

// Get challenge by ID
// (GET /challenges/{challengeID})
func (h *Server) GetChallengesChallengeID(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	var teamID *uuid.UUID
	if user.TeamID != nil {
		teamID = user.TeamID
	}

	detail, err := h.challenge.ChallengeUC.GetDetail(r.Context(), challengeIDParsed, teamID)
	if h.OnError(w, r, err, "GetChallengesChallengeID", "GetDetail") {
		return
	}

	ip := kitMiddleware.GetClientIPFromContext(r.Context())
	reqCtx := r.Context()
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(reqCtx), asyncLogTimeout)
		defer cancel()
		_ = h.user.TrackingUC.TrackChallengeOpen(ctx, user.ID, challengeIDParsed, ip) //nolint:errcheck
	}()

	httputil.RenderOK(w, r, response.FromChallengeDetail(detail))
}

// Get challenge solves
// (GET /challenges/{challengeID}/solves)
func (h *Server) GetChallengesChallengeIDSolves(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	solves, err := h.challenge.ChallengeUC.GetSolves(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDSolves", "GetSolves") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallengeSolves(solves))
}

// Get challenge tags
// (GET /challenges/{challengeID}/tags)
func (h *Server) GetChallengesChallengeIDTags(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	tags, err := h.challenge.TagUC.GetByChallengeID(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDTags", "GetByChallengeID") {
		return
	}
	httputil.RenderOK(w, r, response.FromTagList(tags))
}

// Get challenge types
// (GET /challenges/types)
func (h *Server) GetChallengesTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.challenge.ChallengeUC.GetTypes(r.Context())
	if h.OnError(w, r, err, "GetChallengesTypes", "GetTypes") {
		return
	}
	httputil.RenderOK(w, r, response.FromChallengeTypes(types))
}

// Get challenge requirements
// (GET /challenges/{challengeID}/requirements)
func (h *Server) GetChallengesChallengeIDRequirements(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	requirements, err := h.challenge.ChallengeUC.GetRequirements(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDRequirements", "GetRequirements") {
		return
	}
	httputil.RenderOK(w, r, response.FromChallengeRequirements(requirements))
}

// Get challenge solution
// (GET /challenges/solutions)
func (h *Server) GetChallengesSolutions(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	if !h.checkWriteupEnabled(w, r, "GetChallengesSolutions", "WriteupCheck") {
		return
	}
	if user.TeamID == nil {
		httputil.RenderOK(w, r, response.EmptyChallengeSolutionEntryList())
		return
	}
	team, err := h.team.TeamUC.GetByID(r.Context(), *user.TeamID)
	if h.OnError(w, r, err, "GetChallengesSolutions", "TeamCheck") {
		return
	}
	if team.IsBanned {
		h.OnError(w, r, httperr.ErrTeamBanned, "GetChallengesSolutions", "BanCheck")
		return
	}

	entries, err := h.challenge.ChallengeUC.ListSolutions(r.Context(), *user.TeamID)
	if h.OnError(w, r, err, "GetChallengesSolutions", "ListSolutions") {
		return
	}
	var allFiles []*domain.File
	for _, e := range entries {
		allFiles = append(allFiles, e.Files...)
	}
	urls, err := h.buildDownloadURLs(r.Context(), allFiles, user.TeamID, user.Role == domain.RoleAdmin)
	if h.OnError(w, r, err, "GetChallengesSolutions", "buildDownloadURLs") {
		return
	}
	httputil.RenderOK(w, r, response.FromChallengeSolutionEntryList(entries, urls))
}

// Get challenge solution
// (GET /challenges/{challengeID}/solution)
func (h *Server) GetChallengesChallengeIDSolution(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	if !h.checkWriteupEnabled(w, r, "GetChallengesChallengeIDSolution", "WriteupCheck") {
		return
	}
	if user.TeamID != nil {
		team, err := h.team.TeamUC.GetByID(r.Context(), *user.TeamID)
		if h.OnError(w, r, err, "GetChallengesChallengeIDSolution", "TeamCheck") {
			return
		}
		if team.IsBanned {
			h.OnError(w, r, httperr.ErrTeamBanned, "GetChallengesChallengeIDSolution", "BanCheck")
			return
		}
	}

	solution, err := h.challenge.ChallengeUC.GetSolution(r.Context(), challengeIDParsed, user.TeamID)
	if h.OnError(w, r, err, "GetChallengesChallengeIDSolution", "GetSolution") {
		return
	}
	urls, err := h.buildDownloadURLs(r.Context(), solution.Files, user.TeamID, user.Role == domain.RoleAdmin)
	if h.OnError(w, r, err, "GetChallengesChallengeIDSolution", "buildDownloadURLs") {
		return
	}
	httputil.RenderOK(w, r, response.FromChallengeSolution(solution, urls))
}

// Create or update challenge solution
// (POST /admin/challenges/{challengeID}/solution)
func (h *Server) PostAdminChallengesChallengeIDSolution(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.AdminUpsertSolutionRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	solution, err := h.challenge.ChallengeUC.AdminUpsertSolution(r.Context(), challengeIDParsed, request.AdminUpsertSolutionRequestToParams(&req))
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDSolution", "AdminUpsertSolution") {
		return
	}
	urls, err := h.buildDownloadURLs(r.Context(), solution.Files, nil, true)
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDSolution", "buildDownloadURLs") {
		return
	}
	httputil.RenderOK(w, r, response.FromChallengeSolution(solution, urls))
}

// Delete challenge solution
// (DELETE /admin/challenges/{challengeID}/solution)
func (h *Server) DeleteAdminChallengesChallengeIDSolution(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	err := h.challenge.ChallengeUC.AdminDeleteSolution(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "DeleteAdminChallengesChallengeIDSolution", "AdminDeleteSolution") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// Get challenge flags (admin)
// (GET /admin/challenges/{challengeID}/flags)
func (h *Server) GetAdminChallengesChallengeIDFlags(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	flags, err := h.challenge.ChallengeUC.GetFlags(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetAdminChallengesChallengeIDFlags", "GetFlags") {
		return
	}
	httputil.RenderOK(w, r, response.FromChallengeFlags(flags))
}

// Set challenge requirements (admin)
// (PUT /admin/challenges/{challengeID}/requirements)
func (h *Server) PutAdminChallengesChallengeIDRequirements(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.SetChallengeRequirementsRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	requirementIDs, err := request.ParseUUIDSlice(req.RequirementIds, "requirement_id")
	if h.OnError(w, r, err, "PutAdminChallengesChallengeIDRequirements", "ParseRequirementIDs") {
		return
	}

	err = h.challenge.ChallengeUC.SetRequirements(r.Context(), challengeIDParsed, requirementIDs)
	if h.OnError(w, r, err, "PutAdminChallengesChallengeIDRequirements", "SetRequirements") {
		return
	}

	httputil.RenderNoContent(w, r)
}
