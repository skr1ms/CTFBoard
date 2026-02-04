package v1

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get challenges list
// (GET /challenges)
func (h *Server) GetChallenges(w http.ResponseWriter, r *http.Request, params openapi.GetChallengesParams) {
	user, ok := RequireUser(w, r)
	if !ok {
		return
	}

	var tagID *uuid.UUID
	if params.Tag != nil && *params.Tag != "" {
		if id, err := uuid.Parse(*params.Tag); err == nil {
			tagID = &id
		}
	}

	challenges, err := h.challengeUC.GetAll(r.Context(), user.TeamID, tagID)
	if h.OnError(w, r, err, "GetChallenges", "GetAll") {
		return
	}

	RenderOK(w, r, response.FromChallengeList(challenges))
}

// Submit flag
// (POST /challenges/{ID}/submit)
func (h *Server) PostChallengesIDSubmit(w http.ResponseWriter, r *http.Request, ID string) {
	challengeuuid, ok := ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := DecodeAndValidate[openapi.RequestSubmitFlagRequest](
		w, r, h.validator, h.logger, "PostChallengesIDSubmit",
	)
	if !ok {
		return
	}

	user, ok := RequireUser(w, r)
	if !ok {
		return
	}

	flag := request.SubmitFlagRequestToFlag(&req)
	valid, err := h.challengeUC.SubmitFlag(r.Context(), challengeuuid, flag, user.ID, user.TeamID)

	sub := &entity.Submission{
		UserID:        user.ID,
		ChallengeID:   challengeuuid,
		SubmittedFlag: flag,
		IsCorrect:     valid,
		IP:            GetClientIP(r),
		CreatedAt:     time.Now(),
	}
	if user.TeamID != nil {
		sub.TeamID = user.TeamID
	}
	if logErr := h.submissionUC.LogSubmission(r.Context(), sub); logErr != nil {
		h.logger.WithError(logErr).Error("restapi - v1 - PostChallengesIDSubmit - LogSubmission")
	}

	if h.OnError(w, r, err, "PostChallengesIDSubmit", "SubmitFlag") {
		return
	}

	if !valid {
		RenderError(w, r, http.StatusBadRequest, "invalid flag")
		return
	}

	RenderOK(w, r, map[string]string{"message": "flag accepted"})
}

// Create challenge
// (POST /admin/challenges)
func (h *Server) PostAdminChallenges(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestCreateChallengeRequest](
		w, r, h.validator, h.logger, "PostAdminChallenges",
	)
	if !ok {
		return
	}

	title, desc, cat, pts, initVal, minVal, decay, flag, isHidden, isRegex, isCaseInsens, flagRegex, tagIDs := request.CreateChallengeRequestToParams(&req)
	challenge, err := h.challengeUC.Create(
		r.Context(),
		title, desc, cat, pts, initVal, minVal, decay, flag,
		isHidden, isRegex, isCaseInsens, flagRegex, tagIDs,
	)
	if h.OnError(w, r, err, "PostAdminChallenges", "Create") {
		return
	}

	RenderCreated(w, r, response.FromChallenge(challenge))
}

// Delete challenge
// (DELETE /admin/challenges/{ID})
func (h *Server) DeleteAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeuuid, ok := ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := RequireUser(w, r)
	if !ok {
		return
	}

	clientIP := GetClientIP(r)

	err := h.challengeUC.Delete(r.Context(), challengeuuid, user.ID, clientIP)
	if h.OnError(w, r, err, "DeleteAdminChallengesID", "Delete") {
		return
	}

	RenderNoContent(w, r)
}

// Update challenge
// (PUT /admin/challenges/{ID})
func (h *Server) PutAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeuuid, ok := ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := DecodeAndValidate[openapi.RequestUpdateChallengeRequest](
		w, r, h.validator, h.logger, "PutAdminChallengesID",
	)
	if !ok {
		return
	}

	title, desc, cat, pts, initVal, minVal, decay, flag, isHidden, isRegex, isCaseInsens, flagRegex, tagIDs := request.UpdateChallengeRequestToParams(&req)
	challenge, err := h.challengeUC.Update(
		r.Context(),
		challengeuuid,
		title, desc, cat, pts, initVal, minVal, decay, flag,
		isHidden, isRegex, isCaseInsens, flagRegex, tagIDs,
	)
	if h.OnError(w, r, err, "PutAdminChallengesID", "Update") {
		return
	}

	RenderOK(w, r, response.FromChallenge(challenge))
}
