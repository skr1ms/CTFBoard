package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /admin/users).
func (h *Server) GetAdminUsers(w http.ResponseWriter, r *http.Request, params openapi.GetAdminUsersParams) {
	field, err := request.AdminUsersFieldFromParams(params)
	if h.OnError(w, r, err, "GetAdminUsers", "Field") {
		return
	}

	q, ok := helper.ParseOptionalSearchQuery(w, r, params.Q, maxSearchQueryLen, h.OnError, "GetAdminUsers", "Q")
	if !ok {
		return
	}

	if h.OnError(w, r, request.ValidateAdminUsersSearch(field, q), "GetAdminUsers", "Q") {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.user.UserUC.ListUsers(r.Context(), q, field, page, perPage)
	if h.OnError(w, r, err, "GetAdminUsers", "ListUsers") {
		return
	}

	httputil.RenderOK(w, r, response.FromAdminUserList(result.Data, result.Total, result.Page, result.PerPage))
}

// (POST /admin/users).
func (h *Server) PostAdminUsers(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.AdminCreateUserRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	username, email, password, role, err := request.AdminCreateUserRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminUsers", "AdminCreateUserRequestToParams") {
		return
	}

	user, err := h.user.UserUC.AdminCreate(r.Context(), username, email, password, role)
	if h.OnError(w, r, err, "PostAdminUsers", "AdminCreate") {
		return
	}

	httputil.RenderCreated(w, r, response.FromAdminUser(user))
}

// (PATCH /admin/users/{ID}).
func (h *Server) PatchAdminUsersID(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.AdminUpdateUserRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	username, email, role, password, isVerified, err := request.AdminUpdateUserRequestToParams(&req)
	if h.OnError(w, r, err, "PatchAdminUsersID", "AdminUpdateUserRequestToParams") {
		return
	}

	user, err := h.user.UserUC.AdminUpdate(r.Context(), userIDParsed, username, email, role, password, isVerified)
	if h.OnError(w, r, err, "PatchAdminUsersID", "AdminUpdate") {
		return
	}

	httputil.RenderOK(w, r, response.FromAdminUser(user))
}

// (DELETE /admin/users/{ID}).
func (h *Server) DeleteAdminUsersID(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	actor, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if h.OnError(w, r, h.user.UserUC.AdminDelete(r.Context(), userIDParsed, actor.ID), "DeleteAdminUsersID", "AdminDelete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (POST /admin/users/{ID}/ban).
func (h *Server) PostAdminUsersIDBan(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	actor, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.BanUserRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	if h.OnError(w, r, h.user.UserUC.BanUser(r.Context(), userIDParsed, request.BanUserRequestToParams(&req), actor.ID), "PostAdminUsersIDBan", "BanUser") {
		return
	}

	httputil.RenderOK(w, r, response.Message("user banned"))
}

// (DELETE /admin/users/{ID}/ban).
func (h *Server) DeleteAdminUsersIDBan(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	actor, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if h.OnError(w, r, h.user.UserUC.UnbanUser(r.Context(), userIDParsed, actor.ID), "DeleteAdminUsersIDBan", "UnbanUser") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (GET /admin/users/{ID}/tracking).
func (h *Server) GetAdminUsersIDTracking(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetAdminUsersIDTrackingParams) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.user.TrackingUC.GetByUser(r.Context(), userIDParsed, page, perPage)
	if h.OnError(w, r, err, "GetAdminUsersIDTracking", "GetByUser") {
		return
	}

	httputil.RenderOK(w, r, response.FromTrackingList(result.Data, result.Total, result.Page, result.PerPage))
}

// (GET /admin/users/{ID}/missing-challenges).
func (h *Server) GetAdminUsersIDMissingChallenges(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	challenges, err := h.challenge.AdminUC.GetMissingChallengesByUserID(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetAdminUsersIDMissingChallenges", "GetMissingChallengesByUserID") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallenges(challenges))
}
