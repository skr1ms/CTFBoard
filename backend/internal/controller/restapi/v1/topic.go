package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /admin/topics).
func (h *Server) GetAdminTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := h.challenge.TopicUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetAdminTopics", "GetAll") {
		return
	}

	httputil.RenderOK(w, r, response.FromTopicList(topics))
}

// (POST /admin/topics).
func (h *Server) PostAdminTopics(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.CreateTopicRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	name, err := request.CreateTopicRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminTopics", "CreateTopicRequestToParams") {
		return
	}

	topic, err := h.challenge.TopicUC.Create(r.Context(), name)
	if h.OnError(w, r, err, "PostAdminTopics", "Create") {
		return
	}

	httputil.RenderCreated(w, r, response.FromTopic(topic))
}

// (PUT /admin/topics/{ID}).
func (h *Server) PutAdminTopicsID(w http.ResponseWriter, r *http.Request, ID string) {
	topicIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateTopicRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	name, err := request.UpdateTopicRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminTopicsID", "UpdateTopicRequestToParams") {
		return
	}

	topic, err := h.challenge.TopicUC.Update(r.Context(), topicIDParsed, name)
	if h.OnError(w, r, err, "PutAdminTopicsID", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.FromTopic(topic))
}

// (DELETE /admin/topics/{ID}).
func (h *Server) DeleteAdminTopicsID(w http.ResponseWriter, r *http.Request, ID string) {
	topicIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.challenge.TopicUC.Delete(r.Context(), topicIDParsed), "DeleteAdminTopicsID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (GET /admin/challenges/{challengeID}/topics).
func (h *Server) GetAdminChallengesChallengeIDTopics(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	topics, err := h.challenge.TopicUC.GetByChallengeID(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetAdminChallengesChallengeIDTopics", "GetByChallengeID") {
		return
	}

	httputil.RenderOK(w, r, response.FromTopicList(topics))
}

// (PUT /admin/challenges/{challengeID}/topics).
func (h *Server) PutAdminChallengesChallengeIDTopics(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.SetChallengeTopicsRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	rawTopicIDs, err := request.SetChallengeTopicsRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminChallengesChallengeIDTopics", "SetChallengeTopicsRequestToParams") {
		return
	}

	topicIDs, err := request.ParseUUIDSlice(&rawTopicIDs, "topic_id")
	if h.OnError(w, r, err, "PutAdminChallengesChallengeIDTopics", "ParseTopicIDs") {
		return
	}

	err = h.challenge.TopicUC.SetByChallengeID(r.Context(), challengeIDParsed, topicIDs)
	if h.OnError(w, r, err, "PutAdminChallengesChallengeIDTopics", "SetByChallengeID") {
		return
	}

	httputil.RenderNoContent(w, r)
}
