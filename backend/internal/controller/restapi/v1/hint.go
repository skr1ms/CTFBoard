package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get hints for challenge
// (GET /challenges/{challengeID}/hints)
func (h *Server) GetChallengesChallengeIDHints(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeuuid, ok := ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := RequireUser(w, r)
	if !ok {
		return
	}

	hints, err := h.hintUC.GetByChallengeID(r.Context(), challengeuuid, user.TeamID)
	if h.OnError(w, r, err, "GetChallengesChallengeIDHints", "GetByChallengeID") {
		return
	}

	RenderOK(w, r, response.FromHintWithUnlockList(hints))
}

// Unlock hint
// (POST /challenges/{challengeID}/hints/{hintID}/unlock)
func (h *Server) PostChallengesChallengeIDHintsHintIDUnlock(w http.ResponseWriter, r *http.Request, challengeID, hintID string) {
	hintuuid, ok := ParseUUID(w, r, hintID)
	if !ok {
		return
	}

	user, ok := RequireUser(w, r)
	if !ok {
		return
	}

	if user.TeamID == nil {
		RenderError(w, r, http.StatusBadRequest, "user must be in a team")
		return
	}

	hint, err := h.hintUC.UnlockHint(r.Context(), *user.TeamID, hintuuid)
	if h.OnError(w, r, err, "PostChallengesChallengeIDHintsHintIDUnlock", "UnlockHint") {
		return
	}

	RenderOK(w, r, response.FromUnlockedHint(hint))
}

// Create hint
// (POST /admin/challenges/{challengeID}/hints)
func (h *Server) PostAdminChallengesChallengeIDHints(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeuuid, ok := ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	req, ok := DecodeAndValidate[openapi.RequestCreateHintRequest](
		w, r, h.validator, h.logger, "PostAdminChallengesChallengeIDHints",
	)
	if !ok {
		return
	}

	content, cost, orderIndex := request.CreateHintRequestToParams(&req)
	hint, err := h.hintUC.Create(r.Context(), challengeuuid, content, cost, orderIndex)
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDHints", "Create") {
		return
	}

	RenderCreated(w, r, response.FromHint(hint))
}

// Update hint
// (PUT /admin/hints/{ID})
func (h *Server) PutAdminHintsID(w http.ResponseWriter, r *http.Request, ID string) {
	hintuuid, ok := ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := DecodeAndValidate[openapi.RequestUpdateHintRequest](
		w, r, h.validator, h.logger, "PutAdminHintsID",
	)
	if !ok {
		return
	}

	content, cost, orderIndex := request.UpdateHintRequestToParams(&req)
	hint, err := h.hintUC.Update(r.Context(), hintuuid, content, cost, orderIndex)
	if h.OnError(w, r, err, "PutAdminHintsID", "Update") {
		return
	}

	RenderOK(w, r, response.FromHint(hint))
}

// Delete hint
// (DELETE /admin/hints/{ID})
func (h *Server) DeleteAdminHintsID(w http.ResponseWriter, r *http.Request, ID string) {
	hintuuid, ok := ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.hintUC.Delete(r.Context(), hintuuid), "DeleteAdminHintsID", "Delete") {
		return
	}

	RenderNoContent(w, r)
}
