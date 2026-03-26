package v1

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const defaultUserSortField = "username"

// User login
// (POST /auth/login).
func (h *Server) PostAuthLogin(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.LoginRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	email, password := request.LoginRequestToParams(&req)

	tokenPair, err := h.user.UserUC.Login(r.Context(), email, password)
	if h.OnError(w, r, err, "PostAuthLogin", "Login") {
		return
	}

	httputil.RenderOK(w, r, response.FromTokenPair(tokenPair))
}

// Register new user
// (POST /auth/register).
func (h *Server) PostAuthRegister(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.RegisterRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	username, email, password, customFields, err := request.RegisterRequestToParams(&req)
	if h.OnError(w, r, err, "PostAuthRegister", "RegisterRequestToParams") {
		return
	}

	user, err := h.user.UserUC.Register(r.Context(), username, email, password, customFields)
	if h.OnError(w, r, err, "PostAuthRegister", "Register") {
		return
	}

	if err := h.user.EmailUC.SendVerificationEmail(r.Context(), user); err != nil {
		h.infra.Logger.WithError(err).Warn("restapi - v1 - PostAuthRegister - SendVerificationEmail")
	}

	httputil.RenderCreated(w, r, response.FromUserForRegister(user))
}

// Get current user info
// (GET /auth/me).
func (h *Server) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	httputil.RenderOK(w, r, response.FromUserForMe(user))
}

// Refresh access token
// (POST /auth/refresh).
func (h *Server) PostAuthRefresh(w http.ResponseWriter, r *http.Request, params openapi.PostAuthRefreshParams) {
	refreshToken := params.Authorization
	if t, ok := strings.CutPrefix(refreshToken, "Bearer "); ok {
		refreshToken = t
	}

	if refreshToken == "" {
		h.OnError(w, r, httperr.ErrNotAuthenticated(), "PostAuthRefresh", "MissingToken")

		return
	}

	tokenPair, err := h.infra.JWTService.RefreshTokens(r.Context(), refreshToken)
	if err != nil {
		var mapped error = httperr.ErrNotAuthenticated()

		if errors.Is(err, httperr.ErrUserBanned) {
			mapped = httperr.ErrUserBanned
		}

		h.OnError(w, r, mapped, "PostAuthRefresh", "RefreshTokens")

		return
	}

	httputil.RenderOK(w, r, response.FromTokenPair(tokenPair))
}

// Get user profile
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

// Logout user and revoke refresh token
// (POST /auth/logout).
func (h *Server) PostAuthLogout(w http.ResponseWriter, r *http.Request, params openapi.PostAuthLogoutParams) {
	var refreshToken string

	if params.Authorization != nil {
		if t, ok := strings.CutPrefix(*params.Authorization, "Bearer "); ok {
			refreshToken = t
		}
	}

	if refreshToken == "" && r.Body != nil && r.ContentLength != 0 {
		r.Body = io.NopCloser(io.LimitReader(r.Body, maxLogoutBodySize))

		req, ok := httputil.DecodeAndValidate[openapi.LogoutRequest](
			w, r, h.infra.Validator,
		)
		if !ok {
			return
		}

		if req.RefreshToken != nil {
			refreshToken = *req.RefreshToken
		}
	}

	if refreshToken == "" {
		h.OnError(w, r, httperr.ErrNotAuthenticated(), "PostAuthLogout", "MissingToken")

		return
	}

	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if accessToken, ok := strings.CutPrefix(authHeader, "Bearer "); ok && accessToken != "" {
			err := h.infra.JWTService.RevokeAccessToken(r.Context(), accessToken)
			if err != nil {
				h.infra.Logger.WithError(err).Warn("restapi - v1 - PostAuthLogout - RevokeAccessToken")
			}
		}
	}

	err := h.infra.JWTService.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		h.OnError(w, r, httperr.ErrNotAuthenticated(), "PostAuthLogout", "RevokeRefreshToken")

		return
	}

	httputil.RenderNoContent(w, r)
}

// List users with search and pagination
// (GET /users).
func (h *Server) GetUsers(w http.ResponseWriter, r *http.Request, params openapi.GetUsersParams) {
	q := ""

	if params.Q != nil && *params.Q != "" {
		var ok bool

		q, ok = helper.ParseSearchQuery(w, r, params.Q, maxSearchQueryLen, h.OnError, "GetUsers", "Q")
		if !ok {
			return
		}
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	var search *string

	if q != "" {
		search = &q
	}

	result, err := h.user.UserUC.ListUsers(r.Context(), search, defaultUserSortField, page, perPage)
	if h.OnError(w, r, err, "GetUsers", "ListUsers") {
		return
	}

	httputil.RenderOK(w, r, response.FromUserList(result.Data, result.Total, result.Page, result.PerPage))
}

// Get current user's solves
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

// Get user's solves by user ID
// (GET /users/{ID}/solves).
func (h *Server) GetUsersIDSolves(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	_, ok = helper.RequireUser(w, r)
	if !ok {
		return
	}

	solves, err := h.user.UserUC.GetUserSolves(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUsersIDSolves", "GetUserSolves") {
		return
	}

	httputil.RenderOK(w, r, response.FromSolveWithDetailsList(solves))
}

// Get current user's failed submissions
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

	httputil.RenderOK(w, r, response.FromFailListPublic(fails.Data, fails.Total, fails.Page, fails.PerPage))
}

// Get user's failed submissions by user ID
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

	if user.ID != userIDParsed && user.Role != domain.RoleAdmin {
		h.OnError(w, r, httperr.ErrAccessDenied, "GetUsersIDFails", "AccessCheck")

		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	fails, err := h.user.UserUC.GetUserFails(r.Context(), userIDParsed, page, perPage)
	if h.OnError(w, r, err, "GetUsersIDFails", "GetUserFails") {
		return
	}

	httputil.RenderOK(w, r, response.FromFailListPublic(fails.Data, fails.Total, fails.Page, fails.PerPage))
}

// Get current user's awards
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

// Get user's awards by user ID
// (GET /users/{ID}/awards).
func (h *Server) GetUsersIDAwards(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	_, ok = helper.RequireUser(w, r)
	if !ok {
		return
	}

	awards, err := h.user.UserUC.GetUserAwards(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUsersIDAwards", "GetUserAwards") {
		return
	}

	httputil.RenderOK(w, r, response.FromAwardList(awards))
}

var allowedUserListFields = []string{"username", "ip"}

// List users (admin) with search and pagination
// (GET /admin/users).
func (h *Server) GetAdminUsers(w http.ResponseWriter, r *http.Request, params openapi.GetAdminUsersParams) {
	field := defaultUserSortField

	if params.Field != nil {
		parsed, ok := httputil.ParseEnumQuery(r, "field", allowedUserListFields)
		if !ok {
			h.OnError(w, r, httperr.NewValidationErrorf("field must be one of: username, ip"), "GetAdminUsers", "Field")

			return
		}

		field = string(parsed)
	}

	var q *string

	if params.Q != nil && *params.Q != "" {
		s, ok := helper.ParseSearchQuery(w, r, params.Q, maxSearchQueryLen, h.OnError, "GetAdminUsers", "Q")
		if !ok {
			return
		}

		if field == "ip" && net.ParseIP(*params.Q) == nil {
			h.OnError(w, r, httperr.NewValidationErrorf("invalid IP address"), "GetAdminUsers", "Q")

			return
		}

		q = &s
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.user.UserUC.ListUsers(r.Context(), q, field, page, perPage)
	if h.OnError(w, r, err, "GetAdminUsers", "ListUsers") {
		return
	}

	httputil.RenderOK(w, r, response.FromAdminUserList(result.Data, result.Total, result.Page, result.PerPage))
}

// Create user (admin)
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

// Update user (admin)
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

// Delete user (admin)
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

// Ban user (admin)
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

// Unban user (admin)
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

// Update current user profile
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

// Get current user's submissions
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

// Get user IP tracking (admin)
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

// Get user missing challenges (admin)
// (GET /admin/users/{ID}/missing-challenges).
func (h *Server) GetAdminUsersIDMissingChallenges(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	challenges, err := h.challenge.ChallengeUC.GetMissingChallengesByUserID(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetAdminUsersIDMissingChallenges", "GetMissingChallengesByUserID") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallenges(challenges))
}
