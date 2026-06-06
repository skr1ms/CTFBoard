package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /users/{ID}).
func (h *Server) GetUsersID(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	profile, err := h.user.UserUC.GetProfile(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUsersID", "GetProfile") {
		return
	}

	httputil.RenderOK(w, r, response.FromUserProfile(profile))
}

// (GET /users).
func (h *Server) GetUsers(w http.ResponseWriter, r *http.Request, params openapi.GetUsersParams) {
	search, ok := helper.ParseOptionalSearchQuery(w, r, params.Q, maxSearchQueryLen, h.OnError, "GetUsers", "Q")
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.user.UserUC.ListUsers(r.Context(), search, request.UserSearchFieldUsername, page, perPage)
	if h.OnError(w, r, err, "GetUsers", "ListUsers") {
		return
	}

	httputil.RenderOK(w, r, response.FromUserList(result.Data, result.Total, result.Page, result.PerPage))
}

// (GET /users/me/solves).
func (h *Server) GetUsersMeSolves(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	solves, err := h.user.UserUC.GetUserSolves(r.Context(), user.ID)
	if h.OnError(w, r, err, "GetUsersMeSolves", "GetUserSolves") {
		return
	}

	httputil.RenderOK(w, r, response.FromSolveWithDetailsList(solves))
}

// (GET /users/{ID}/solves).
func (h *Server) GetUsersIDSolves(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.UserMatchesOrAdmin(user, userIDParsed) {
		h.OnError(w, r, helper.ErrAccessDenied, "GetUsersIDSolves", "AccessCheck")

		return
	}

	solves, err := h.user.UserUC.GetUserSolves(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUsersIDSolves", "GetUserSolves") {
		return
	}

	httputil.RenderOK(w, r, response.FromSolveWithDetailsList(solves))
}

// (GET /users/me/fails).
func (h *Server) GetUsersMeFails(w http.ResponseWriter, r *http.Request, params openapi.GetUsersMeFailsParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	fails, err := h.user.UserUC.GetUserFails(r.Context(), user.ID, page, perPage)
	if h.OnError(w, r, err, "GetUsersMeFails", "GetUserFails") {
		return
	}

	httputil.RenderOK(w, r, response.FromFailListSelf(fails.Data, fails.Total, fails.Page, fails.PerPage))
}

// (GET /users/{ID}/fails).
func (h *Server) GetUsersIDFails(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetUsersIDFailsParams) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.UserMatchesOrAdmin(user, userIDParsed) {
		h.OnError(w, r, helper.ErrAccessDenied, "GetUsersIDFails", "AccessCheck")

		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	fails, err := h.user.UserUC.GetUserFails(r.Context(), userIDParsed, page, perPage)
	if h.OnError(w, r, err, "GetUsersIDFails", "GetUserFails") {
		return
	}

	httputil.RenderOK(w, r, response.FromFailListPublic(fails.Data, fails.Total, fails.Page, fails.PerPage))
}

// (GET /users/me/awards).
func (h *Server) GetUsersMeAwards(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	awards, err := h.user.UserUC.GetUserAwards(r.Context(), user.ID)
	if h.OnError(w, r, err, "GetUsersMeAwards", "GetUserAwards") {
		return
	}

	httputil.RenderOK(w, r, response.FromAwardList(awards))
}

// (GET /users/{ID}/awards).
func (h *Server) GetUsersIDAwards(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.UserMatchesOrAdmin(user, userIDParsed) {
		h.OnError(w, r, helper.ErrAccessDenied, "GetUsersIDAwards", "AccessCheck")

		return
	}

	awards, err := h.user.UserUC.GetUserAwards(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUsersIDAwards", "GetUserAwards") {
		return
	}

	httputil.RenderOK(w, r, response.FromAwardList(awards))
}

// (PATCH /auth/me).
func (h *Server) PatchAuthMe(w http.ResponseWriter, r *http.Request) {
	me, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateProfileRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	username, email, currentPassword, newPassword := request.UpdateProfileRequestToParams(&req)

	user, err := h.user.UserUC.UpdateProfile(r.Context(), me.ID, username, email, currentPassword, newPassword)
	if h.OnError(w, r, err, "PatchAuthMe", "UpdateProfile") {
		return
	}

	httputil.RenderOK(w, r, response.FromUserForMe(user))
}

// (GET /users/me/submissions).
func (h *Server) GetUsersMeSubmissions(w http.ResponseWriter, r *http.Request, params openapi.GetUsersMeSubmissionsParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.user.UserUC.GetMySubmissions(r.Context(), user.ID, page, perPage)
	if h.OnError(w, r, err, "GetUsersMeSubmissions", "GetMySubmissions") {
		return
	}

	httputil.RenderOK(w, r, response.FromSubmissionListPublic(result.Data, result.Total, result.Page, result.PerPage))
}
