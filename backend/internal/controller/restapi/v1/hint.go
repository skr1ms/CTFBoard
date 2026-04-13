package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /challenges/{challengeID}/hints).
func (h *Server) GetChallengesChallengeIDHints(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
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

	httputil.RenderOK(w, r, response.FromHintWithUnlockList(hints))
}

// (POST /challenges/{challengeID}/hints/{hintID}/unlock).
func (h *Server) PostChallengesChallengeIDHintsHintIDUnlock(w http.ResponseWriter, r *http.Request, challengeID, hintID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	hintIDParsed, ok := httputil.ParseUUID(w, r, hintID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	teamID, ok := helper.RequireTeamID(w, r, user, h.OnError, "PostChallengesChallengeIDHintsHintIDUnlock")
	if !ok {
		return
	}

	hint, err := h.challenge.HintUC.UnlockHint(r.Context(), user.ID, teamID, challengeIDParsed, hintIDParsed)
	if h.OnError(w, r, err, "PostChallengesChallengeIDHintsHintIDUnlock", "UnlockHint") {
		return
	}

	httputil.RenderOK(w, r, response.FromUnlockedHint(hint))
}

// (POST /admin/challenges/{challengeID}/hints).
func (h *Server) PostAdminChallengesChallengeIDHints(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.CreateHintRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	title, content, cost, orderIndex, err := request.CreateHintRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDHints", "CreateHintRequestToParams") {
		return
	}

	hint, err := h.challenge.HintUC.Create(r.Context(), challengeIDParsed, title, content, cost, orderIndex)
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDHints", "Create") {
		return
	}

	httputil.RenderCreated(w, r, response.FromHint(hint))
}

// (PUT /admin/hints/{ID}).
func (h *Server) PutAdminHintsID(w http.ResponseWriter, r *http.Request, ID string) {
	hintIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateHintRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	title, content, cost, orderIndex, err := request.UpdateHintRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminHintsID", "UpdateHintRequestToParams") {
		return
	}

	hint, err := h.challenge.HintUC.Update(r.Context(), hintIDParsed, title, content, cost, orderIndex)
	if h.OnError(w, r, err, "PutAdminHintsID", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.FromHint(hint))
}

// (DELETE /admin/hints/{ID}).
func (h *Server) DeleteAdminHintsID(w http.ResponseWriter, r *http.Request, ID string) {
	hintIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.challenge.HintUC.Delete(r.Context(), hintIDParsed), "DeleteAdminHintsID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (GET /admin/unlocks).
func (h *Server) GetAdminUnlocks(w http.ResponseWriter, r *http.Request, params openapi.GetAdminUnlocksParams) {
	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.challenge.HintUC.GetAllUnlocks(r.Context(), page, perPage)
	if h.OnError(w, r, err, "GetAdminUnlocks", "GetAll") {
		return
	}

	httputil.RenderOK(w, r, response.FromHintUnlockList(result.Data, result.Total, result.Page, result.PerPage))
}
