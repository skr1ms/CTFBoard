package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
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
	helper.RenderOK(w, r, response.FromTagList(tags))
}

// Create tag
// (POST /admin/tags)
func (h *Server) PostAdminTags(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.CreateTagRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminTags",
	)
	if !ok {
		return
	}
	name, color := request.CreateTagRequestToParams(&req)
	tag, err := h.challenge.TagUC.Create(r.Context(), name, color)
	if h.OnError(w, r, err, "PostAdminTags", "Create") {
		return
	}
	helper.RenderCreated(w, r, response.FromTag(tag))
}

// Update tag
// (PUT /admin/tags/{ID})
func (h *Server) PutAdminTagsID(w http.ResponseWriter, r *http.Request, ID string) {
	tagIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.UpdateTagRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PutAdminTagsID",
	)
	if !ok {
		return
	}
	name, color := request.UpdateTagRequestToParams(&req)
	tag, err := h.challenge.TagUC.Update(r.Context(), tagIDParsed, name, color)
	if h.OnError(w, r, err, "PutAdminTagsID", "Update") {
		return
	}
	helper.RenderOK(w, r, response.FromTag(tag))
}

// Delete tag
// (DELETE /admin/tags/{ID})
func (h *Server) DeleteAdminTagsID(w http.ResponseWriter, r *http.Request, ID string) {
	tagIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	if h.OnError(w, r, h.challenge.TagUC.Delete(r.Context(), tagIDParsed), "DeleteAdminTagsID", "Delete") {
		return
	}
	helper.RenderNoContent(w, r)
}
