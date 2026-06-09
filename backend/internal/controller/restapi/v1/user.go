package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func (h *Server) requirePublicUserVisible(w http.ResponseWriter, r *http.Request, userID uuid.UUID, op string) bool {
	target, err := h.user.UserUC.GetByID(r.Context(), userID)
	if h.OnError(w, r, err, op, "GetByID") {
		return false
	}

	viewer, _ := helper.CurrentUser(r)
	if !helper.UserPublicVisibleToViewer(target, viewer) {
		h.OnError(w, r, helper.ErrUserNotFound, op, "BlockedUser")

		return false
	}

	return true
}

// (GET /users/{ID}).
func (h *Server) GetUsersID(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if !h.requirePublicUserVisible(w, r, userIDParsed, "GetUsersID") {
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

	if !h.requirePublicUserVisible(w, r, userIDParsed, "GetUsersIDSolves") {
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

	if !h.requirePublicUserVisible(w, r, userIDParsed, "GetUsersIDFails") {
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

	if !h.requirePublicUserVisible(w, r, userIDParsed, "GetUsersIDAwards") {
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

	params := request.UpdateProfileRequestToParams(me.ID, &req)

	result, err := h.user.UserUC.UpdateProfile(r.Context(), params)
	if h.OnError(w, r, err, "PatchAuthMe", "UpdateProfile") {
		return
	}

	if result != nil && result.TokenPair != nil {
		h.setRefreshCookie(w, result.TokenPair.RefreshToken)
	}

	httputil.RenderOK(w, r, response.FromUpdateProfileResult(result))
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
