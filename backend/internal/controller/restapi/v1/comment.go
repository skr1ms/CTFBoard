package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Get comments for challenge
// (GET /challenges/{challengeID}/comments)
func (h *Server) GetChallengesChallengeIDComments(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	list, err := h.challenge.CommentUC.GetByChallengeID(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDComments", "GetByChallengeID") {
		return
	}
	helper.RenderOK(w, r, response.FromCommentList(list))
}

// Create comment
// (POST /challenges/{challengeID}/comments)
func (h *Server) PostChallengesChallengeIDComments(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.CreateCommentRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostChallengesChallengeIDComments",
	)
	if !ok {
		return
	}
	content, err := request.CreateCommentRequestToParams(&req)
	if h.OnError(w, r, err, "PostChallengesChallengeIDComments", "CreateCommentRequestToParams") {
		return
	}
	comment, err := h.challenge.CommentUC.Create(r.Context(), user.ID, challengeIDParsed, content)
	if h.OnError(w, r, err, "PostChallengesChallengeIDComments", "Create") {
		return
	}
	helper.RenderCreated(w, r, response.FromComment(comment))
}

// Delete comment
// (DELETE /comments/{ID})
func (h *Server) DeleteCommentsID(w http.ResponseWriter, r *http.Request, ID string) {
	commentIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	if h.OnError(w, r, h.challenge.CommentUC.Delete(r.Context(), commentIDParsed, user.ID, user.Role == entity.RoleAdmin), "DeleteCommentsID", "Delete") {
		return
	}
	helper.RenderNoContent(w, r)
}
