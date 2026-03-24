package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

// Put rating for challenge
// (PUT /challenges/{challengeID}/rating)
func (h *Server) PutChallengesChallengeIDRating(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	if user.TeamID == nil {
		h.OnError(w, r, httperr.ErrUserNotInTeam, "PutChallengesChallengeIDRating", "RequireTeam")
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
	rating, err := h.challenge.RatingUC.PutRating(r.Context(), challengeIDParsed, user.ID, *user.TeamID, value, review)
	if h.OnError(w, r, err, "PutChallengesChallengeIDRating", "PutRating") {
		return
	}
	httputil.RenderOK(w, r, response.FromRating(rating))
}

// Get ratings for challenge
// (GET /challenges/{challengeID}/ratings)
func (h *Server) GetChallengesChallengeIDRatings(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	list, err := h.challenge.RatingUC.GetRatingsByChallengeID(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDRatings", "GetRatingsByChallengeID") {
		return
	}
	httputil.RenderOK(w, r, response.FromRatingList(list))
}
