package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/google/uuid"
)

// Get scoreboard
// (GET /scoreboard)
func (h *Server) GetScoreboard(w http.ResponseWriter, r *http.Request, params openapi.GetScoreboardParams) {
	var bracketID *uuid.UUID
	if params.Bracket != nil {
		u := *params.Bracket
		bracketID = &u
	}
	entries, err := h.comp.SolveUC.GetScoreboard(r.Context(), bracketID)
	if h.OnError(w, r, err, "GetScoreboard", "GetScoreboard") {
		return
	}
	helper.RenderOK(w, r, response.FromScoreboardList(entries))
}

// Get first blood
// (GET /challenges/{ID}/first-blood)
func (h *Server) GetChallengesIDFirstBlood(w http.ResponseWriter, r *http.Request, ID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	entry, err := h.comp.SolveUC.GetFirstBlood(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesIDFirstBlood", "GetFirstBlood") {
		return
	}

	helper.RenderOK(w, r, response.FromFirstBlood(entry))
}
