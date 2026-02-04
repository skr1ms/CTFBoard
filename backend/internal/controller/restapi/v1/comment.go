package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get comments for challenge
// (GET /challenges/{challengeID}/comments)
func (h *Server) GetChallengesChallengeIDComments(w http.ResponseWriter, r *http.Request, challengeID string) {
	cid, ok := ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	list, err := h.commentUC.GetByChallengeID(r.Context(), cid)
	if h.OnError(w, r, err, "GetChallengesChallengeIDComments", "GetByChallengeID") {
		return
	}
	RenderOK(w, r, response.FromCommentList(list))
}

// Create comment
// (POST /challenges/{challengeID}/comments)
func (h *Server) PostChallengesChallengeIDComments(w http.ResponseWriter, r *http.Request, challengeID string) {
	user, ok := RequireUser(w, r)
	if !ok {
		return
	}
	cid, ok := ParseUUID(w, r, challengeID)
	if !ok {
		return
	}
	req, ok := DecodeAndValidate[openapi.RequestCreateCommentRequest](w, r, h.validator, h.logger, "PostChallengesChallengeIDComments")
	if !ok {
		return
	}
	comment, err := h.commentUC.Create(r.Context(), user.ID, cid, req.Content)
	if h.OnError(w, r, err, "PostChallengesChallengeIDComments", "Create") {
		return
	}
	RenderCreated(w, r, response.FromComment(comment))
}

// Delete comment
// (DELETE /comments/{ID})
func (h *Server) DeleteCommentsID(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := RequireUser(w, r)
	if !ok {
		return
	}
	commentID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	if h.OnError(w, r, h.commentUC.Delete(r.Context(), commentID, user.ID), "DeleteCommentsID", "Delete") {
		return
	}
	RenderNoContent(w, r)
}
