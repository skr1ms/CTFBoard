package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get tags list
// (GET /tags)
func (h *Server) GetTags(w http.ResponseWriter, r *http.Request) {
	h.Handle("GetTags", h.handleGetTags)(w, r)
}

func (h *Server) handleGetTags(w http.ResponseWriter, r *http.Request) (HandlerResult, error) {
	tags, err := h.challenge.TagUC.GetAll(r.Context())
	if err != nil {
		return HandlerResult{}, err
	}
	return OK(response.FromTagList(tags)), nil
}

// Create tag
// (POST /admin/tags)
func (h *Server) PostAdminTags(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RequestCreateTagRequest](w, r, h.infra.Validator, h.infra.Logger, "PostAdminTags")
	if !ok {
		return
	}
	color := ""
	if req.Color != nil {
		color = *req.Color
	}
	tag, err := h.challenge.TagUC.Create(r.Context(), req.Name, color)
	if h.OnError(w, r, err, "PostAdminTags", "Create") {
		return
	}
	helper.RenderCreated(w, r, response.FromTag(tag))
}

// Update tag
// (PUT /admin/tags/{ID})
func (h *Server) PutAdminTagsID(w http.ResponseWriter, r *http.Request, id string) {
	tagID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.RequestUpdateTagRequest](w, r, h.infra.Validator, h.infra.Logger, "PutAdminTagsID")
	if !ok {
		return
	}
	color := ""
	if req.Color != nil {
		color = *req.Color
	}
	tag, err := h.challenge.TagUC.Update(r.Context(), tagID, req.Name, color)
	if h.OnError(w, r, err, "PutAdminTagsID", "Update") {
		return
	}
	helper.RenderOK(w, r, response.FromTag(tag))
}

// Delete tag
// (DELETE /admin/tags/{ID})
func (h *Server) DeleteAdminTagsID(w http.ResponseWriter, r *http.Request, id string) {
	tagID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}
	if h.OnError(w, r, h.challenge.TagUC.Delete(r.Context(), tagID), "DeleteAdminTagsID", "Delete") {
		return
	}
	helper.RenderNoContent(w, r)
}
