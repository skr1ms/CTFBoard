package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get published pages list
// (GET /pages)
func (h *Server) GetPages(w http.ResponseWriter, r *http.Request) {
	list, err := h.pageUC.GetPublishedList(r.Context())
	if h.OnError(w, r, err, "GetPages", "GetPublishedList") {
		return
	}
	RenderOK(w, r, response.FromPageList(list))
}

// Get page by slug
// (GET /pages/{slug})
func (h *Server) GetPagesSlug(w http.ResponseWriter, r *http.Request, slug string) {
	page, err := h.pageUC.GetBySlug(r.Context(), slug)
	if h.OnError(w, r, err, "GetPagesSlug", "GetBySlug") {
		return
	}
	RenderOK(w, r, response.FromPage(page))
}

// Get all pages (admin)
// (GET /admin/pages)
func (h *Server) GetAdminPages(w http.ResponseWriter, r *http.Request) {
	list, err := h.pageUC.GetAllList(r.Context())
	if h.OnError(w, r, err, "GetAdminPages", "GetAllList") {
		return
	}
	RenderOK(w, r, response.FromPageFullList(list))
}

// Create page
// (POST /admin/pages)
func (h *Server) PostAdminPages(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestCreatePageRequest](w, r, h.validator, h.logger, "PostAdminPages")
	if !ok {
		return
	}
	content := ""
	if req.Content != nil {
		content = *req.Content
	}
	isDraft := true
	if req.IsDraft != nil {
		isDraft = *req.IsDraft
	}
	orderIndex := 0
	if req.OrderIndex != nil {
		orderIndex = *req.OrderIndex
	}
	page, err := h.pageUC.Create(r.Context(), req.Title, req.Slug, content, isDraft, orderIndex)
	if h.OnError(w, r, err, "PostAdminPages", "Create") {
		return
	}
	RenderCreated(w, r, response.FromPage(page))
}

// Get page by ID (admin)
// (GET /admin/pages/{ID})
func (h *Server) GetAdminPagesID(w http.ResponseWriter, r *http.Request, id string) {
	pageID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	page, err := h.pageUC.GetByID(r.Context(), pageID)
	if h.OnError(w, r, err, "GetAdminPagesID", "GetByID") {
		return
	}
	RenderOK(w, r, response.FromPage(page))
}

// Update page
// (PUT /admin/pages/{ID})
func (h *Server) PutAdminPagesID(w http.ResponseWriter, r *http.Request, id string) {
	pageID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	req, ok := DecodeAndValidate[openapi.RequestUpdatePageRequest](w, r, h.validator, h.logger, "PutAdminPagesID")
	if !ok {
		return
	}
	content := ""
	if req.Content != nil {
		content = *req.Content
	}
	isDraft := false
	if req.IsDraft != nil {
		isDraft = *req.IsDraft
	}
	orderIndex := 0
	if req.OrderIndex != nil {
		orderIndex = *req.OrderIndex
	}
	page, err := h.pageUC.Update(r.Context(), pageID, req.Title, req.Slug, content, isDraft, orderIndex)
	if h.OnError(w, r, err, "PutAdminPagesID", "Update") {
		return
	}
	RenderOK(w, r, response.FromPage(page))
}

// Delete page
// (DELETE /admin/pages/{ID})
func (h *Server) DeleteAdminPagesID(w http.ResponseWriter, r *http.Request, id string) {
	pageID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	if h.OnError(w, r, h.pageUC.Delete(r.Context(), pageID), "DeleteAdminPagesID", "Delete") {
		return
	}
	RenderNoContent(w, r)
}
