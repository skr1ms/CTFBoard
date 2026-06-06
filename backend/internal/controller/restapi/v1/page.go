package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /pages).
func (h *Server) GetPages(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.PageUC.GetPublishedList(r.Context())
	if h.OnError(w, r, err, "GetPages", "GetPublishedList") {
		return
	}

	setPublicCache(w, cacheStatic, false)
	httputil.RenderOK(w, r, response.FromPageList(list))
}

// (GET /pages/{slug}).
func (h *Server) GetPagesSlug(w http.ResponseWriter, r *http.Request, slug string) {
	page, err := h.admin.PageUC.GetBySlug(r.Context(), slug)
	if h.OnError(w, r, err, "GetPagesSlug", "GetBySlug") {
		return
	}

	setPublicCache(w, cacheStatic, false)
	httputil.RenderOK(w, r, response.FromPage(page))
}

// (GET /admin/pages).
func (h *Server) GetAdminPages(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.PageUC.GetAllList(r.Context())
	if h.OnError(w, r, err, "GetAdminPages", "GetAllList") {
		return
	}

	httputil.RenderOK(w, r, response.FromPageFullList(list))
}

// (POST /admin/pages).
func (h *Server) PostAdminPages(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.CreatePageRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
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
