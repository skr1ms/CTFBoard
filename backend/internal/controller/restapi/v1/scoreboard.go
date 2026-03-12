package v1

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Get scoreboard
// (GET /scoreboard)
func (h *Server) GetScoreboard(w http.ResponseWriter, r *http.Request, params openapi.GetScoreboardParams) {
	var bracketID *uuid.UUID
	if params.Bracket != nil {
		u := *params.Bracket
		bracketID = &u
	}
	forceLive := params.Live != nil && *params.Live
	if forceLive {
		if user, ok := middleware.GetUser(r.Context()); !ok || user.Role != entity.RoleAdmin {
			forceLive = false
		}
	}
	entries, err := h.comp.SolveUC.GetScoreboard(r.Context(), bracketID, forceLive)
	if h.OnError(w, r, err, "GetScoreboard", "GetScoreboard") {
		return
	}
	helper.RenderOK(w, r, response.FromScoreboardList(entries))
}

// Get first blood
// (GET /challenges/{ID}/first-blood)
func (h *Server) GetChallengesChallengeIDFirstBlood(w http.ResponseWriter, r *http.Request, challengeID string, params openapi.GetChallengesChallengeIDFirstBloodParams) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	forceLive := params.Live != nil && *params.Live
	if forceLive {
		if user, ok := middleware.GetUser(r.Context()); !ok || user.Role != entity.RoleAdmin {
			forceLive = false
		}
	}
	entry, err := h.comp.SolveUC.GetFirstBlood(r.Context(), challengeIDParsed, forceLive)
	if h.OnError(w, r, err, "GetChallengesChallengeIDFirstBlood", "GetFirstBlood") {
		return
	}
	helper.RenderOK(w, r, response.FromFirstBlood(entry))
}
