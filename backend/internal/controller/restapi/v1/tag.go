package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get tags list
// (GET /tags)
func (h *Server) GetTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.tagUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetTags", "GetAll") {
		return
	}
	RenderOK(w, r, response.FromTagList(tags))
}

// Create tag
// (POST /admin/tags)
func (h *Server) PostAdminTags(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestCreateTagRequest](w, r, h.validator, h.logger, "PostAdminTags")
	if !ok {
		return
	}
	color := ""
	if req.Color != nil {
		color = *req.Color
	}
	tag, err := h.tagUC.Create(r.Context(), req.Name, color)
	if h.OnError(w, r, err, "PostAdminTags", "Create") {
		return
	}
	RenderCreated(w, r, response.FromTag(tag))
}

// Update tag
// (PUT /admin/tags/{ID})
func (h *Server) PutAdminTagsID(w http.ResponseWriter, r *http.Request, id string) {
	tagID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	req, ok := DecodeAndValidate[openapi.RequestUpdateTagRequest](w, r, h.validator, h.logger, "PutAdminTagsID")
	if !ok {
		return
	}
	color := ""
	if req.Color != nil {
		color = *req.Color
	}
	tag, err := h.tagUC.Update(r.Context(), tagID, req.Name, color)
	if h.OnError(w, r, err, "PutAdminTagsID", "Update") {
		return
	}
	RenderOK(w, r, response.FromTag(tag))
}

// Delete tag
// (DELETE /admin/tags/{ID})
func (h *Server) DeleteAdminTagsID(w http.ResponseWriter, r *http.Request, id string) {
	tagID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	if h.OnError(w, r, h.tagUC.Delete(r.Context(), tagID), "DeleteAdminTagsID", "Delete") {
		return
	}
	RenderNoContent(w, r)
}
