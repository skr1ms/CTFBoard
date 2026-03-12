package v1

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const defaultUserSortField = "username"

// User login
// (POST /auth/login)
func (h *Server) PostAuthLogin(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.LoginRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAuthLogin",
	)
	if !ok {
		return
	}

	email, password := request.LoginRequestToParams(&req)
	tokenPair, err := h.user.UserUC.Login(r.Context(), email, password)
	if h.OnError(w, r, err, "PostAuthLogin", "Login") {
		return
	}

	helper.RenderOK(w, r, response.FromTokenPair(tokenPair))
}

// Register new user
// (POST /auth/register)
func (h *Server) PostAuthRegister(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RegisterRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAuthRegister",
	)
	if !ok {
		return
	}

	username, email, password, customFields := request.RegisterRequestToParams(&req)
	user, err := h.user.UserUC.Register(r.Context(), username, email, password, customFields)
	if h.OnError(w, r, err, "PostAuthRegister", "Register") {
		return
	}

	if err := h.user.EmailUC.SendVerificationEmail(r.Context(), user); err != nil {
		h.infra.Logger.WithError(err).Warn("restapi - v1 - PostAuthRegister - SendVerificationEmail")
	}

	helper.RenderCreated(w, r, response.FromUserForRegister(user))
}

// Get current user info
// (GET /auth/me)
func (h *Server) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	helper.RenderOK(w, r, response.FromUserForMe(user))
}

// Refresh access token
// (POST /auth/refresh)
func (h *Server) PostAuthRefresh(w http.ResponseWriter, r *http.Request, params openapi.PostAuthRefreshParams) {
	refreshToken := params.Authorization
	if t, ok := strings.CutPrefix(refreshToken, "Bearer "); ok {
		refreshToken = t
	}
	if refreshToken == "" {
		h.OnError(w, r, helper.ErrNotAuthenticated, "PostAuthRefresh", "MissingToken")
		return
	}
	tokenPair, err := h.infra.JWTService.RefreshTokens(r.Context(), refreshToken)
	if err != nil {
		mapped := helper.ErrNotAuthenticated
		if errors.Is(err, helper.ErrUserBanned) {
			mapped = helper.ErrUserBanned
		}
		h.OnError(w, r, mapped, "PostAuthRefresh", "RefreshTokens")
		return
	}
	helper.RenderOK(w, r, response.FromTokenPair(tokenPair))
}

// Get user profile
// (GET /users/{ID})
func (h *Server) GetUsersID(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	profile, err := h.user.UserUC.GetProfile(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUsersID", "GetProfile") {
		return
	}

	helper.RenderOK(w, r, response.FromUserProfile(profile))
}

// Logout user and revoke refresh token
// (POST /auth/logout)
func (h *Server) PostAuthLogout(w http.ResponseWriter, r *http.Request, params openapi.PostAuthLogoutParams) {
	var refreshToken string
	if params.Authorization != nil {
		if t, ok := strings.CutPrefix(*params.Authorization, "Bearer "); ok {
			refreshToken = t
		}
	}
	if refreshToken == "" && r.Body != nil && r.ContentLength != 0 {
		r.Body = io.NopCloser(io.LimitReader(r.Body, 4096))
		req, ok := helper.DecodeAndValidate[openapi.LogoutRequest](
			w, r, h.infra.Validator, h.infra.Logger, "PostAuthLogout",
		)
		if !ok {
			return
		}
		if req.RefreshToken != nil {
			refreshToken = *req.RefreshToken
		}
	}
	if refreshToken == "" {
		h.OnError(w, r, helper.ErrNotAuthenticated, "PostAuthLogout", "MissingToken")
		return
	}
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if accessToken, ok := strings.CutPrefix(authHeader, "Bearer "); ok && accessToken != "" {
			if err := h.infra.JWTService.RevokeAccessToken(r.Context(), accessToken); err != nil {
				h.infra.Logger.WithError(err).Warn("restapi - v1 - PostAuthLogout - RevokeAccessToken")
			}
		}
	}
	if err := h.infra.JWTService.RevokeRefreshToken(r.Context(), refreshToken); err != nil {
		h.OnError(w, r, helper.ErrNotAuthenticated, "PostAuthLogout", "RevokeRefreshToken")
		return
	}
	helper.RenderNoContent(w, r)
}

// List users with search and pagination
// (GET /users)
func (h *Server) GetUsers(w http.ResponseWriter, r *http.Request, params openapi.GetUsersParams) {
	q := ""
	if params.Q != nil && *params.Q != "" {
		if !helper.ValidateSearchQ(*params.Q) {
			h.OnError(w, r, helper.NewValidationErrorf("invalid search query"), "GetUsers", "Q")
			return
		}
		q = helper.SanitizeSearchQ(*params.Q, 100)
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

	helper.RenderOK(w, r, response.FromUserList(result.Data, result.Total, result.Page, result.PerPage))
}

// Get current user's solves
// (GET /users/me/solves)
func (h *Server) GetUsersMeSolves(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	solves, err := h.user.UserUC.GetUserSolves(r.Context(), user.ID)
	if h.OnError(w, r, err, "GetUsersMeSolves", "GetUserSolves") {
		return
	}

	helper.RenderOK(w, r, response.FromSolveWithDetailsList(solves))
}

// Get user's solves by user ID
// (GET /users/{ID}/solves)
func (h *Server) GetUsersIDSolves(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	if user.ID != userIDParsed && user.Role != entity.RoleAdmin {
		h.OnError(w, r, helper.ErrAccessDenied, "GetUsersIDSolves", "AccessCheck")
		return
	}

	solves, err := h.user.UserUC.GetUserSolves(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUsersIDSolves", "GetUserSolves") {
		return
	}

	helper.RenderOK(w, r, response.FromSolveWithDetailsList(solves))
}

// Get current user's failed submissions
// (GET /users/me/fails)
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

	helper.RenderOK(w, r, response.FromFailListPublic(fails.Data, fails.Total, fails.Page, fails.PerPage))
}

// Get user's failed submissions by user ID
// (GET /users/{ID}/fails)
func (h *Server) GetUsersIDFails(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetUsersIDFailsParams) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	if user.ID != userIDParsed && user.Role != entity.RoleAdmin {
		h.OnError(w, r, helper.ErrAccessDenied, "GetUsersIDFails", "AccessCheck")
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	fails, err := h.user.UserUC.GetUserFails(r.Context(), userIDParsed, page, perPage)
	if h.OnError(w, r, err, "GetUsersIDFails", "GetUserFails") {
		return
	}

	helper.RenderOK(w, r, response.FromFailListPublic(fails.Data, fails.Total, fails.Page, fails.PerPage))
}

// Get current user's awards
// (GET /users/me/awards)
func (h *Server) GetUsersMeAwards(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	awards, err := h.user.UserUC.GetUserAwards(r.Context(), user.ID)
	if h.OnError(w, r, err, "GetUsersMeAwards", "GetUserAwards") {
		return
	}

	helper.RenderOK(w, r, response.FromAwardList(awards))
}

// Get user's awards by user ID
// (GET /users/{ID}/awards)
func (h *Server) GetUsersIDAwards(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	if user.ID != userIDParsed && user.Role != entity.RoleAdmin {
		h.OnError(w, r, helper.ErrAccessDenied, "GetUsersIDAwards", "AccessCheck")
		return
	}

	awards, err := h.user.UserUC.GetUserAwards(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUsersIDAwards", "GetUserAwards") {
		return
	}

	helper.RenderOK(w, r, response.FromAwardList(awards))
}

// List users (admin) with search and pagination
// (GET /admin/users)
var allowedUserListFields = map[string]bool{"username": true, "ip": true}

func (h *Server) GetAdminUsers(w http.ResponseWriter, r *http.Request, params openapi.GetAdminUsersParams) {
	field := defaultUserSortField
	if params.Field != nil {
		field = string(*params.Field)
	}
	if !allowedUserListFields[field] {
		h.OnError(w, r, helper.NewValidationErrorf("field must be one of: username, ip"), "GetAdminUsers", "Field")
		return
	}
	var q *string
	if params.Q != nil && *params.Q != "" {
		raw := *params.Q
		if !helper.ValidateSearchQ(raw) {
			h.OnError(w, r, helper.NewValidationErrorf("invalid search query"), "GetAdminUsers", "Q")
			return
		}
		if field == "ip" && net.ParseIP(raw) == nil {
			h.OnError(w, r, helper.NewValidationErrorf("invalid IP address"), "GetAdminUsers", "Q")
			return
		}
		s := helper.SanitizeSearchQ(raw, 100)
		q = &s
	}
	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.user.UserUC.ListUsers(r.Context(), q, field, page, perPage)
	if h.OnError(w, r, err, "GetAdminUsers", "ListUsers") {
		return
	}

	helper.RenderOK(w, r, response.FromAdminUserList(result.Data, result.Total, result.Page, result.PerPage))
}

// Create user (admin)
// (POST /admin/users)
func (h *Server) PostAdminUsers(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.AdminCreateUserRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminUsers",
	)
	if !ok {
		return
	}

	username, email, password, role := request.AdminCreateUserRequestToParams(&req)
	user, err := h.user.UserUC.AdminCreate(r.Context(), username, email, password, role)
	if h.OnError(w, r, err, "PostAdminUsers", "AdminCreate") {
		return
	}

	helper.RenderCreated(w, r, response.FromAdminUser(user))
}

// Update user (admin)
// (PATCH /admin/users/{ID})
func (h *Server) PatchAdminUsersID(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.AdminUpdateUserRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PatchAdminUsersID",
	)
	if !ok {
		return
	}

	username, email, role, password, isVerified := request.AdminUpdateUserRequestToParams(&req)
	user, err := h.user.UserUC.AdminUpdate(r.Context(), userIDParsed, username, email, role, password, isVerified)
	if h.OnError(w, r, err, "PatchAdminUsersID", "AdminUpdate") {
		return
	}

	helper.RenderOK(w, r, response.FromAdminUser(user))
}

// Delete user (admin)
// (DELETE /admin/users/{ID})
func (h *Server) DeleteAdminUsersID(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
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

	helper.RenderNoContent(w, r)
}

// Ban user (admin)
// (POST /admin/users/{ID}/ban)
func (h *Server) PostAdminUsersIDBan(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	actor, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.BanUserRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminUsersIDBan",
	)
	if !ok {
		return
	}

	if h.OnError(w, r, h.user.UserUC.BanUser(r.Context(), userIDParsed, request.BanUserRequestToParams(&req), actor.ID), "PostAdminUsersIDBan", "BanUser") {
		return
	}

	helper.RenderOK(w, r, response.Message("user banned"))
}

// Unban user (admin)
// (DELETE /admin/users/{ID}/ban)
func (h *Server) DeleteAdminUsersIDBan(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
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

	helper.RenderNoContent(w, r)
}

// Update current user profile
// (PATCH /auth/me)
func (h *Server) PatchAuthMe(w http.ResponseWriter, r *http.Request) {
	me, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.UpdateProfileRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PatchAuthMe",
	)
	if !ok {
		return
	}

	username, email, currentPassword, newPassword := request.UpdateProfileRequestToParams(&req)
	user, err := h.user.UserUC.UpdateProfile(r.Context(), me.ID, username, email, currentPassword, newPassword)
	if h.OnError(w, r, err, "PatchAuthMe", "UpdateProfile") {
		return
	}

	helper.RenderOK(w, r, response.FromUserForMe(user))
}

// Get current user's submissions
// (GET /users/me/submissions)
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

	helper.RenderOK(w, r, response.FromSubmissionListPublic(result.Data, result.Total, result.Page, result.PerPage))
}

// Get user IP tracking (admin)
// (GET /admin/users/{ID}/tracking)
func (h *Server) GetAdminUsersIDTracking(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetAdminUsersIDTrackingParams) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)
	result, err := h.user.TrackingUC.GetByUser(r.Context(), userIDParsed, page, perPage)
	if h.OnError(w, r, err, "GetAdminUsersIDTracking", "GetByUser") {
		return
	}
	helper.RenderOK(w, r, response.FromTrackingList(result.Data, result.Total, result.Page, result.PerPage))
}

// Get user missing challenges (admin)
// (GET /admin/users/{ID}/missing-challenges)
func (h *Server) GetAdminUsersIDMissingChallenges(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	challenges, err := h.challenge.ChallengeUC.GetMissingChallengesByUserID(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetAdminUsersIDMissingChallenges", "GetMissingChallengesByUserID") {
		return
	}
	helper.RenderOK(w, r, response.FromChallengeEntityList(challenges))
}
