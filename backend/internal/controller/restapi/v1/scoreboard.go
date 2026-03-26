package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Get scoreboard
// (GET /scoreboard).
func (h *Server) GetScoreboard(w http.ResponseWriter, r *http.Request, params openapi.GetScoreboardParams) {
	var bracketID *uuid.UUID

	if params.Bracket != nil {
		u := *params.Bracket
		bracketID = &u
	}

	forceLive := params.Live != nil && *params.Live
	if forceLive {
		if user, ok := middleware.GetUser(r.Context()); !ok || user.Role != domain.RoleAdmin {
			forceLive = false
		}
	}

	entries, err := h.comp.SolveUC.GetScoreboard(r.Context(), bracketID, forceLive)
	if h.OnError(w, r, err, "GetScoreboard", "GetScoreboard") {
		return
	}

	resp, err := response.FromScoreboardListWithAvatars(r.Context(), entries, h.user.AvatarUC)
	if h.OnError(w, r, err, "GetScoreboard", "FromScoreboardListWithAvatars") {
		return
	}

	httputil.RenderOK(w, r, resp)
}

// Get first blood
// (GET /challenges/{ID}/first-blood).
func (h *Server) GetChallengesChallengeIDFirstBlood(w http.ResponseWriter, r *http.Request, challengeID string, params openapi.GetChallengesChallengeIDFirstBloodParams) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	forceLive := params.Live != nil && *params.Live
	if forceLive {
		if user, ok := middleware.GetUser(r.Context()); !ok || user.Role != domain.RoleAdmin {
			forceLive = false
		}
	}

	entry, err := h.comp.SolveUC.GetFirstBlood(r.Context(), challengeIDParsed, forceLive)
	if h.OnError(w, r, err, "GetChallengesChallengeIDFirstBlood", "GetFirstBlood") {
		return
	}

	httputil.RenderOK(w, r, response.FromFirstBlood(entry))
}
