package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/google/uuid"
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
		id, ok := helper.ParseUUIDField(w, r, *params.Tag, "tag")
		if !ok {
			return
		}
		tagID = &id
	}

	challenges, err := h.challenge.ChallengeUC.GetAll(r.Context(), user.TeamID, tagID)
	if h.OnError(w, r, err, "GetChallenges", "GetAll") {
		return
	}

	helper.RenderOK(w, r, response.FromChallengeList(challenges))
}

// Submit flag
// (POST /challenges/{ID}/submit)
func (h *Server) PostChallengesIDSubmit(w http.ResponseWriter, r *http.Request, ID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.SubmitFlagRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostChallengesIDSubmit",
	)
	if !ok {
		return
	}

	flag := request.SubmitFlagRequestToParams(&req)
	valid, submitErr := h.challenge.ChallengeUC.SubmitFlag(r.Context(), challengeIDParsed, flag, user.ID, user.TeamID)

	isCorrectForLog := valid
	sub := &entity.Submission{
		UserID:        user.ID,
		ChallengeID:   challengeIDParsed,
		SubmittedFlag: flag,
		IsCorrect:     isCorrectForLog,
		IP:            helper.GetClientIP(r, h.infra.TrustedProxyCIDRs),
		CreatedAt:     time.Now(),
	}
	if user.TeamID != nil {
		sub.TeamID = user.TeamID
	}
	if h.comp.SubmissionBatcher != nil {
		h.comp.SubmissionBatcher.Enqueue(sub)
	} else {
		go func() {
			logCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), asyncLogTimeout)
			defer cancel()
			if logErr := h.comp.SubmissionUC.LogSubmission(logCtx, sub); logErr != nil {
				h.infra.Logger.WithError(logErr).Error("restapi - v1 - PostChallengesIDSubmit - LogSubmission")
			}
		}()
	}

	if h.OnError(w, r, submitErr, "PostChallengesIDSubmit", "SubmitFlag") {
		return
	}

	if !valid {
		helper.RenderOK(w, r, response.FromSubmitFlag(false, "incorrect flag"))
		return
	}

	helper.RenderOK(w, r, response.FromSubmitFlag(true, "flag accepted"))
}

// Create challenge
// (POST /admin/challenges)
func (h *Server) PostAdminChallenges(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.CreateChallengeRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminChallenges",
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
		params.IsHidden, params.IsRegex, params.IsCaseInsensitive, params.FlagFormatRegex, params.TagIDs,
	)
	if h.OnError(w, r, err, "PostAdminChallenges", "Create") {
		return
	}

	helper.RenderCreated(w, r, response.FromChallenge(challenge))
}

// Delete challenge
// (DELETE /admin/challenges/{ID})
func (h *Server) DeleteAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	clientIP := helper.GetClientIP(r, h.infra.TrustedProxyCIDRs)

	err := h.challenge.ChallengeUC.Delete(r.Context(), challengeIDParsed, user.ID, clientIP)
	if h.OnError(w, r, err, "DeleteAdminChallengesID", "Delete") {
		return
	}

	helper.RenderNoContent(w, r)
}

// Update challenge
// (PUT /admin/challenges/{ID})
func (h *Server) PutAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.UpdateChallengeRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PutAdminChallengesID",
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
		params.IsHidden, params.IsRegex, params.IsCaseInsensitive, params.FlagFormatRegex, params.TagIDs,
	)
	if h.OnError(w, r, err, "PutAdminChallengesID", "Update") {
		return
	}

	helper.RenderOK(w, r, response.FromChallenge(challenge))
}

// Get challenge by ID
// (GET /challenges/{challengeID})
func (h *Server) GetChallengesChallengeID(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
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

	ip := helper.GetClientIP(r, h.infra.TrustedProxyCIDRs)
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), asyncLogTimeout)
		defer cancel()
		_ = h.user.TrackingUC.TrackChallengeOpen(ctx, user.ID, challengeIDParsed, ip) //nolint:errcheck
	}()

	helper.RenderOK(w, r, response.FromChallengeDetail(detail))
}

// Get challenge solves
// (GET /challenges/{challengeID}/solves)
func (h *Server) GetChallengesChallengeIDSolves(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	solves, err := h.challenge.ChallengeUC.GetSolves(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDSolves", "GetSolves") {
		return
	}

	helper.RenderOK(w, r, response.FromChallengeSolves(solves))
}

// Get challenge tags
// (GET /challenges/{challengeID}/tags)
func (h *Server) GetChallengesChallengeIDTags(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	tags, err := h.challenge.TagUC.GetByChallengeID(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDTags", "GetByChallengeID") {
		return
	}
	helper.RenderOK(w, r, response.FromTagList(tags))
}

// Get challenge types
// (GET /challenges/types)
func (h *Server) GetChallengesTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.challenge.ChallengeUC.GetTypes(r.Context())
	if h.OnError(w, r, err, "GetChallengesTypes", "GetTypes") {
		return
	}
	helper.RenderOK(w, r, response.FromChallengeTypes(types))
}

// Get challenge requirements
// (GET /challenges/{challengeID}/requirements)
func (h *Server) GetChallengesChallengeIDRequirements(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	requirements, err := h.challenge.ChallengeUC.GetRequirements(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDRequirements", "GetRequirements") {
		return
	}
	helper.RenderOK(w, r, response.FromChallengeRequirements(requirements))
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
		helper.RenderOK(w, r, response.EmptyChallengeSolutionEntryList())
		return
	}
	team, err := h.team.TeamUC.GetByID(r.Context(), *user.TeamID)
	if h.OnError(w, r, err, "GetChallengesSolutions", "TeamCheck") {
		return
	}
	if team.IsBanned {
		h.OnError(w, r, helper.ErrTeamBanned, "GetChallengesSolutions", "BanCheck")
		return
	}

	entries, err := h.challenge.ChallengeUC.ListSolutions(r.Context(), *user.TeamID)
	if h.OnError(w, r, err, "GetChallengesSolutions", "ListSolutions") {
		return
	}

	helper.RenderOK(w, r, response.FromChallengeSolutionEntryList(entries, func(e *entity.ChallengeSolutionEntry) map[string]string {
		return h.buildDownloadURLs(r.Context(), e.Files)
	}))
}

// Get challenge solution
// (GET /challenges/{challengeID}/solution)
func (h *Server) GetChallengesChallengeIDSolution(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
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
			h.OnError(w, r, helper.ErrTeamBanned, "GetChallengesChallengeIDSolution", "BanCheck")
			return
		}
	}

	solution, err := h.challenge.ChallengeUC.GetSolution(r.Context(), challengeIDParsed, user.TeamID)
	if h.OnError(w, r, err, "GetChallengesChallengeIDSolution", "GetSolution") {
		return
	}

	helper.RenderOK(w, r, response.FromChallengeSolution(solution, h.buildDownloadURLs(r.Context(), solution.Files)))
}

// Create or update challenge solution
// (POST /admin/challenges/{challengeID}/solution)
func (h *Server) PostAdminChallengesChallengeIDSolution(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.AdminUpsertSolutionRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminChallengesChallengeIDSolution",
	)
	if !ok {
		return
	}

	solution, err := h.challenge.ChallengeUC.AdminUpsertSolution(r.Context(), challengeIDParsed, request.AdminUpsertSolutionRequestToParams(&req))
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDSolution", "AdminUpsertSolution") {
		return
	}

	helper.RenderOK(w, r, response.FromChallengeSolution(solution, h.buildDownloadURLs(r.Context(), solution.Files)))
}

// Delete challenge solution
// (DELETE /admin/challenges/{challengeID}/solution)
func (h *Server) DeleteAdminChallengesChallengeIDSolution(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	err := h.challenge.ChallengeUC.AdminDeleteSolution(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "DeleteAdminChallengesChallengeIDSolution", "AdminDeleteSolution") {
		return
	}

	helper.RenderNoContent(w, r)
}

// Get challenge flags (admin)
// (GET /admin/challenges/{challengeID}/flags)
func (h *Server) GetAdminChallengesChallengeIDFlags(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	flags, err := h.challenge.ChallengeUC.GetFlags(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetAdminChallengesChallengeIDFlags", "GetFlags") {
		return
	}
	helper.RenderOK(w, r, response.FromChallengeFlags(flags))
}

// Set challenge requirements (admin)
// (PUT /admin/challenges/{challengeID}/requirements)
func (h *Server) PutAdminChallengesChallengeIDRequirements(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.SetChallengeRequirementsRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PutAdminChallengesChallengeIDRequirements",
	)
	if !ok {
		return
	}

	requirementIDs, err := request.ParseRequirementIDs(req.RequirementIds)
	if h.OnError(w, r, err, "PutAdminChallengesChallengeIDRequirements", "ParseRequirementIDs") {
		return
	}

	err = h.challenge.ChallengeUC.SetRequirements(r.Context(), challengeIDParsed, requirementIDs)
	if h.OnError(w, r, err, "PutAdminChallengesChallengeIDRequirements", "SetRequirements") {
		return
	}

	helper.RenderNoContent(w, r)
}
