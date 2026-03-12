package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Get hints for challenge
// (GET /challenges/{challengeID}/hints)
func (h *Server) GetChallengesChallengeIDHints(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	hints, err := h.challenge.HintUC.GetByChallengeID(r.Context(), challengeIDParsed, user.TeamID)
	if h.OnError(w, r, err, "GetChallengesChallengeIDHints", "GetByChallengeID") {
		return
	}

	helper.RenderOK(w, r, response.FromHintWithUnlockList(hints))
}

// Unlock hint
// (POST /challenges/{challengeID}/hints/{hintID}/unlock)
func (h *Server) PostChallengesChallengeIDHintsHintIDUnlock(w http.ResponseWriter, r *http.Request, challengeID, hintID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	hintIDParsed, ok := helper.ParseUUID(w, r, hintID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if user.TeamID == nil {
		h.OnError(w, r, helper.ErrUserNotInTeam, "PostChallengesChallengeIDHintsHintIDUnlock", "RequireTeam")
		return
	}

	hint, err := h.challenge.HintUC.UnlockHint(r.Context(), user.ID, *user.TeamID, challengeIDParsed, hintIDParsed)
	if h.OnError(w, r, err, "PostChallengesChallengeIDHintsHintIDUnlock", "UnlockHint") {
		return
	}

	helper.RenderOK(w, r, response.FromUnlockedHint(hint))
}

// Create hint
// (POST /admin/challenges/{challengeID}/hints)
func (h *Server) PostAdminChallengesChallengeIDHints(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.CreateHintRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminChallengesChallengeIDHints",
	)
	if !ok {
		return
	}

	content, cost, orderIndex, err := request.CreateHintRequestToParams(&req)
	if err != nil {
		h.OnError(w, r, err, "PostAdminChallengesChallengeIDHints", "CreateHintRequestToParams")
		return
	}
	hint, err := h.challenge.HintUC.Create(r.Context(), challengeIDParsed, content, cost, orderIndex)
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDHints", "Create") {
		return
	}

	helper.RenderCreated(w, r, response.FromHint(hint))
}

// Update hint
// (PUT /admin/hints/{ID})
func (h *Server) PutAdminHintsID(w http.ResponseWriter, r *http.Request, ID string) {
	hintIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.UpdateHintRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PutAdminHintsID",
	)
	if !ok {
		return
	}

	content, cost, orderIndex, err := request.UpdateHintRequestToParams(&req)
	if err != nil {
		h.OnError(w, r, err, "PutAdminHintsID", "UpdateHintRequestToParams")
		return
	}
	hint, err := h.challenge.HintUC.Update(r.Context(), hintIDParsed, content, cost, orderIndex)
	if h.OnError(w, r, err, "PutAdminHintsID", "Update") {
		return
	}

	helper.RenderOK(w, r, response.FromHint(hint))
}

// Delete hint
// (DELETE /admin/hints/{ID})
func (h *Server) DeleteAdminHintsID(w http.ResponseWriter, r *http.Request, ID string) {
	hintIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.challenge.HintUC.Delete(r.Context(), hintIDParsed), "DeleteAdminHintsID", "Delete") {
		return
	}

	helper.RenderNoContent(w, r)
}

// Get all hint unlocks (admin)
// (GET /admin/unlocks)
func (h *Server) GetAdminUnlocks(w http.ResponseWriter, r *http.Request, params openapi.GetAdminUnlocksParams) {
	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.challenge.HintUC.GetAllUnlocks(r.Context(), page, perPage)
	if h.OnError(w, r, err, "GetAdminUnlocks", "GetAll") {
		return
	}

	helper.RenderOK(w, r, response.FromHintUnlockList(result.Data, result.Total, result.Page, result.PerPage))
}
