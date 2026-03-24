package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Get tags list
// (GET /tags)
func (h *Server) GetTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.challenge.TagUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetTags", "GetAll") {
		return
	}
	httputil.RenderOK(w, r, response.FromTagList(tags))
}

// Create tag
// (POST /admin/tags)
func (h *Server) PostAdminTags(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.CreateTagRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}
	if err := request.ValidateCreateTagRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PostAdminTags", "Validate") {
		return
	}
	name, color, err := request.CreateTagRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminTags", "CreateTagRequestToParams") {
		return
	}
	tag, err := h.challenge.TagUC.Create(r.Context(), name, color)
	if h.OnError(w, r, err, "PostAdminTags", "Create") {
		return
	}
	httputil.RenderCreated(w, r, response.FromTag(tag))
}

// Update tag
// (PUT /admin/tags/{ID})
func (h *Server) PutAdminTagsID(w http.ResponseWriter, r *http.Request, ID string) {
	tagIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	req, ok := httputil.DecodeAndValidate[openapi.UpdateTagRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}
	if err := request.ValidateUpdateTagRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PutAdminTagsID", "Validate") {
		return
	}
	name, color, err := request.UpdateTagRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminTagsID", "UpdateTagRequestToParams") {
		return
	}
	tag, err := h.challenge.TagUC.Update(r.Context(), tagIDParsed, name, color)
	if h.OnError(w, r, err, "PutAdminTagsID", "Update") {
		return
	}
	httputil.RenderOK(w, r, response.FromTag(tag))
}

// Delete tag
// (DELETE /admin/tags/{ID})
func (h *Server) DeleteAdminTagsID(w http.ResponseWriter, r *http.Request, ID string) {
	tagIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	if h.OnError(w, r, h.challenge.TagUC.Delete(r.Context(), tagIDParsed), "DeleteAdminTagsID", "Delete") {
		return
	}
	httputil.RenderNoContent(w, r)
}
