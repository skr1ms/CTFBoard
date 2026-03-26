package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	slugpkg "github.com/TakuyaYagam1/AstroCTFb/pkg/slug"
)

// Get published pages list
// (GET /pages).
func (h *Server) GetPages(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.PageUC.GetPublishedList(r.Context())
	if h.OnError(w, r, err, "GetPages", "GetPublishedList") {
		return
	}

	httputil.RenderOK(w, r, response.FromPageList(list))
}

// Get page by slug
// (GET /pages/{slug}).
func (h *Server) GetPagesSlug(w http.ResponseWriter, r *http.Request, slug string) {
	if slug == "" || !slugpkg.MatchPageSlug(slug) {
		h.OnError(w, r, httperr.ErrPageSlugInvalid, "GetPagesSlug", "ValidateSlug")

		return
	}

	page, err := h.admin.PageUC.GetBySlug(r.Context(), slug)
	if h.OnError(w, r, err, "GetPagesSlug", "GetBySlug") {
		return
	}

	httputil.RenderOK(w, r, response.FromPage(page))
}

// Get all pages (admin)
// (GET /admin/pages).
func (h *Server) GetAdminPages(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.PageUC.GetAllList(r.Context())
	if h.OnError(w, r, err, "GetAdminPages", "GetAllList") {
		return
	}

	httputil.RenderOK(w, r, response.FromPageFullList(list))
}

// Create page
// (POST /admin/pages).
func (h *Server) PostAdminPages(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.CreatePageRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	if err := request.ValidateCreatePageRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PostAdminPages", "Validate") {
		return
	}

	title, slug, content, isDraft, orderIndex, err := request.CreatePageRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminPages", "CreatePageRequestToParams") {
		return
	}

	page, err := h.admin.PageUC.Create(r.Context(), title, slug, content, isDraft, orderIndex)
	if h.OnError(w, r, err, "PostAdminPages", "Create") {
		return
	}

	httputil.RenderCreated(w, r, response.FromPage(page))
}

// Get page by ID (admin)
// (GET /admin/pages/{ID}).
func (h *Server) GetAdminPagesID(w http.ResponseWriter, r *http.Request, ID string) {
	pageID, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	page, err := h.admin.PageUC.GetByID(r.Context(), pageID)
	if h.OnError(w, r, err, "GetAdminPagesID", "GetByID") {
		return
	}

	httputil.RenderOK(w, r, response.FromPage(page))
}

// Update page
// (PUT /admin/pages/{ID}).
func (h *Server) PutAdminPagesID(w http.ResponseWriter, r *http.Request, ID string) {
	pageID, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdatePageRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	if err := request.ValidateUpdatePageRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PutAdminPagesID", "Validate") {
		return
	}

	title, slug, content, isDraft, orderIndex, err := request.UpdatePageRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminPagesID", "UpdatePageRequestToParams") {
		return
	}

	page, err := h.admin.PageUC.Update(r.Context(), pageID, title, slug, content, isDraft, orderIndex)
	if h.OnError(w, r, err, "PutAdminPagesID", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.FromPage(page))
}

// Delete page
// (DELETE /admin/pages/{ID}).
func (h *Server) DeleteAdminPagesID(w http.ResponseWriter, r *http.Request, ID string) {
	pageID, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.admin.PageUC.Delete(r.Context(), pageID), "DeleteAdminPagesID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}
