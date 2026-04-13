package v1

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /scoreboard).
func (h *Server) GetScoreboard(w http.ResponseWriter, r *http.Request, params openapi.GetScoreboardParams) {
	var bracketID *uuid.UUID

	if params.Bracket != nil {
		u := *params.Bracket
		bracketID = &u
	}

	forceLive := forceLiveFromParams(r, params.Live)

	entries, err := h.comp.SolveUC.GetScoreboard(r.Context(), bracketID, forceLive)
	if h.OnError(w, r, err, "GetScoreboard", "GetScoreboard") {
		return
	}

	thumbURLs, err := resolveTeamAvatarThumbs(r.Context(), entries, h)
	if h.OnError(w, r, err, "GetScoreboard", "resolveTeamAvatarThumbs") {
		return
	}

	httputil.RenderOK(w, r, response.FromScoreboardListWithAvatars(entries, thumbURLs))
}

// (GET /challenges/{ID}/first-blood).
func (h *Server) GetChallengesChallengeIDFirstBlood(w http.ResponseWriter, r *http.Request, challengeID string, params openapi.GetChallengesChallengeIDFirstBloodParams) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	forceLive := forceLiveFromParams(r, params.Live)

	entry, err := h.comp.SolveUC.GetFirstBlood(r.Context(), challengeIDParsed, forceLive)
	if h.OnError(w, r, err, "GetChallengesChallengeIDFirstBlood", "GetFirstBlood") {
		return
	}

	httputil.RenderOK(w, r, response.FromFirstBlood(entry))
}

func resolveTeamAvatarThumbs(ctx context.Context, entries []*domain.ScoreboardEntry, h *Server) (map[uuid.UUID]string, error) {
	if h.user.AvatarUC == nil {
		return nil, nil
	}

	teamIDs := make([]uuid.UUID, len(entries))
	for i, e := range entries {
		teamIDs[i] = e.TeamID
	}

	return h.user.AvatarUC.GetTeamAvatarURLBatch(ctx, teamIDs)
}
