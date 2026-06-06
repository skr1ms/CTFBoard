package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /challenges/{challengeID}/comments).
func (h *Server) GetChallengesChallengeIDComments(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	list, err := h.challenge.CommentUC.GetByChallengeID(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDComments", "GetByChallengeID") {
		return
	}

	httputil.RenderOK(w, r, response.FromCommentList(list))
}

// (POST /challenges/{challengeID}/comments).
func (h *Server) PostChallengesChallengeIDComments(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.CreateCommentRequest](
		w, r, h.infra.Validator,
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

	httputil.RenderCreated(w, r, response.FromComment(comment))
}

// (DELETE /comments/{ID}).
func (h *Server) DeleteCommentsID(w http.ResponseWriter, r *http.Request, ID string) {
	commentIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if h.OnError(w, r, h.challenge.CommentUC.Delete(r.Context(), commentIDParsed, user.ID, helper.IsAdmin(user)), "DeleteCommentsID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}
