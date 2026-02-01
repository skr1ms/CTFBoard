package v1

import (
	"net/http"

	"github.com/google/uuid"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

// Get hints for challenge
// (GET /challenges/{challengeID}/hints)
func (h *Server) GetChallengesChallengeIDHints(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeuuid, err := uuid.Parse(challengeID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	hints, err := h.hintUC.GetByChallengeID(r.Context(), challengeuuid, user.TeamID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetChallengesChallengeIDHints")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, response.FromHintWithUnlockList(hints))
}

// Unlock hint
// (POST /challenges/{challengeID}/hints/{hintID}/unlock)
func (h *Server) PostChallengesChallengeIDHintsHintIDUnlock(w http.ResponseWriter, r *http.Request, challengeID, hintID string) {
	hintuuid, err := uuid.Parse(hintID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	if user.TeamID == nil {
		httputil.RenderError(w, r, http.StatusBadRequest, "user must be in a team")
		return
	}

	hint, err := h.hintUC.UnlockHint(r.Context(), *user.TeamID, hintuuid)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostChallengesChallengeIDHintsHintIDUnlock")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, response.FromUnlockedHint(hint))
}

// Create hint
// (POST /admin/challenges/{challengeID}/hints)
func (h *Server) PostAdminChallengesChallengeIDHints(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeuuid, err := uuid.Parse(challengeID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	req, ok := httputil.DecodeAndValidate[request.CreateHintRequest](
		w, r, h.validator, h.logger, "PostAdminChallengesChallengeIDHints",
	)
	if !ok {
		return
	}

	hint, err := h.hintUC.Create(r.Context(), challengeuuid, req.Content, req.Cost, req.OrderIndex)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAdminChallengesChallengeIDHints")
		handleError(w, r, err)
		return
	}

	httputil.RenderCreated(w, r, response.FromHint(hint))
}

// Update hint
// (PUT /admin/hints/{ID})
func (h *Server) PutAdminHintsID(w http.ResponseWriter, r *http.Request, ID string) {
	hintuuid, err := uuid.Parse(ID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	req, ok := httputil.DecodeAndValidate[request.UpdateHintRequest](
		w, r, h.validator, h.logger, "PutAdminHintsID",
	)
	if !ok {
		return
	}

	hint, err := h.hintUC.Update(r.Context(), hintuuid, req.Content, req.Cost, req.OrderIndex)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PutAdminHintsID")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, response.FromHint(hint))
}

// Delete hint
// (DELETE /admin/hints/{ID})
func (h *Server) DeleteAdminHintsID(w http.ResponseWriter, r *http.Request, ID string) {
	hintuuid, err := uuid.Parse(ID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	if err := h.hintUC.Delete(r.Context(), hintuuid); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - DeleteAdminHintsID")
		handleError(w, r, err)
		return
	}

	httputil.RenderNoContent(w, r)
}
