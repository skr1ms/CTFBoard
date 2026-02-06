package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get comments for challenge
// (GET /challenges/{challengeID}/comments)
func (h *Server) GetChallengesChallengeIDComments(w http.ResponseWriter, r *http.Request, challengeID string) {
	cid, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	list, err := h.challenge.CommentUC.GetByChallengeID(r.Context(), cid)
	if h.OnError(w, r, err, "GetChallengesChallengeIDComments", "GetByChallengeID") {
		return
	}
	helper.RenderOK(w, r, response.FromCommentList(list))
}

// Create comment
// (POST /challenges/{challengeID}/comments)
func (h *Server) PostChallengesChallengeIDComments(w http.ResponseWriter, r *http.Request, challengeID string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	cid, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.RequestCreateCommentRequest](w, r, h.infra.Validator, h.infra.Logger, "PostChallengesChallengeIDComments")
	if !ok {
		return
	}
	comment, err := h.challenge.CommentUC.Create(r.Context(), user.ID, cid, req.Content)
	if h.OnError(w, r, err, "PostChallengesChallengeIDComments", "Create") {
		return
	}
	helper.RenderCreated(w, r, response.FromComment(comment))
}

// Delete comment
// (DELETE /comments/{ID})
func (h *Server) DeleteCommentsID(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	commentID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}
	if h.OnError(w, r, h.challenge.CommentUC.Delete(r.Context(), commentID, user.ID), "DeleteCommentsID", "Delete") {
		return
	}
	helper.RenderNoContent(w, r)
}
