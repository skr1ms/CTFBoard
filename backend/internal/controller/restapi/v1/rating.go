package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (PUT /challenges/{challengeID}/rating).
func (h *Server) PutChallengesChallengeIDRating(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	teamID, ok := helper.RequireTeamID(w, r, user, h.OnError, "PutChallengesChallengeIDRating")
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.PutChallengeRatingRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	value, review, err := request.PutChallengeRatingRequestToParams(&req)
	if h.OnError(w, r, err, "PutChallengesChallengeIDRating", "RequestConversion") {
		return
	}

	rating, err := h.challenge.RatingUC.PutRating(r.Context(), challengeIDParsed, user.ID, teamID, value, review)
	if h.OnError(w, r, err, "PutChallengesChallengeIDRating", "PutRating") {
		return
	}

	httputil.RenderOK(w, r, response.FromRating(rating))
}

// (GET /challenges/{challengeID}/ratings).
func (h *Server) GetChallengesChallengeIDRatings(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	list, err := h.challenge.RatingUC.GetRatingsByChallengeID(r.Context(), challengeIDParsed, user.TeamID)
	if h.OnError(w, r, err, "GetChallengesChallengeIDRatings", "GetRatingsByChallengeID") {
		return
	}

	httputil.RenderOK(w, r, response.FromRatingList(list))
}
