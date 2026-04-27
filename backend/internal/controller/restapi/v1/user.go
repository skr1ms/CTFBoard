package v1

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	refreshCookieName    = "ctf_refresh"
	refreshCookieMaxAge  = 72 * 60 * 60 // 72 h - matches jwtkit RefreshTTL
	defaultUserSortField = "username"
	sortFieldIP          = "ip"
)

func (h *Server) setRefreshCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.user.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Server) clearRefreshCookie(w http.ResponseWriter) {
	h.setRefreshCookie(w, "", -1)
}

// requireOwnUserOrAdmin writes ErrAccessDenied and returns false when the authenticated
// user is neither the target user nor an admin. Pass the handler and step names for
// consistent error logging.
func (h *Server) requireOwnUserOrAdmin(w http.ResponseWriter, r *http.Request, user *domain.User, targetID uuid.UUID, handler, step string) bool {
	if user.ID == targetID || helper.IsAdmin(user) {
		return true
	}

	h.OnError(w, r, apperr.ErrAccessDenied, handler, step)

	return false
}

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

	h.setRefreshCookie(w, tokenPair.RefreshToken, refreshCookieMaxAge)
	httputil.RenderOK(w, r, response.FromTokenPair(tokenPair))
}

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

// (GET /auth/me).
func (h *Server) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	resp := response.FromUserForMe(user)

	if user.IsBanned && resp.BanStatus != nil {
		canAppeal, hasPending, err := h.user.AppealUC.CanAppeal(r.Context(), user.ID)
		if h.OnError(w, r, err, "GetAuthMe", "CanAppeal") {
			return
		}

		resp.BanStatus.CanAppeal = &canAppeal
		resp.BanStatus.HasPendingAppeal = &hasPending
	}

	httputil.RenderOK(w, r, resp)
}

// PostAuthRefresh issues a new token pair from the httpOnly refresh cookie.
// Error mapping is intentional: ErrUserBanned is preserved so clients can
// display a ban message; all other JWTService errors are mapped to
// ErrNotAuthenticated to avoid leaking token state to the caller.
// (POST /auth/refresh).
func (h *Server) PostAuthRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		h.OnError(w, r, apperr.ErrNotAuthenticated, "PostAuthRefresh", "MissingRefreshCookie")

		return
	}

	tokenPair, err := h.infra.JWTService.RefreshTokens(r.Context(), cookie.Value)
	if err != nil {
		var mapped = apperr.ErrNotAuthenticated

		if errors.Is(err, apperr.ErrUserBanned) {
			mapped = apperr.ErrUserBanned
		}

		h.OnError(w, r, mapped, "PostAuthRefresh", "RefreshTokens")

		return
	}

	h.setRefreshCookie(w, tokenPair.RefreshToken, refreshCookieMaxAge)
	httputil.RenderOK(w, r, response.FromTokenPair(tokenPair))
}

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

// PostAuthLogout revokes the refresh token from the httpOnly cookie and
// optionally revokes the current access token from the Authorization header.
// (POST /auth/logout).
func (h *Server) PostAuthLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		h.OnError(w, r, apperr.ErrNotAuthenticated, "PostAuthLogout", "MissingRefreshCookie")

		return
	}

	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if accessToken, ok := strings.CutPrefix(authHeader, "Bearer "); ok && accessToken != "" {
			if revokeErr := h.infra.JWTService.RevokeAccessToken(r.Context(), accessToken); revokeErr != nil {
				h.infra.Logger.WithError(revokeErr).Warn("restapi - v1 - PostAuthLogout - RevokeAccessToken")
			}
		}
	}

	if err := h.infra.JWTService.RevokeRefreshToken(r.Context(), cookie.Value); err != nil {
		h.OnError(w, r, apperr.ErrNotAuthenticated, "PostAuthLogout", "RevokeRefreshToken")

		return
	}

	h.clearRefreshCookie(w)
	httputil.RenderNoContent(w, r)
}

// (GET /users).
func (h *Server) GetUsers(w http.ResponseWriter, r *http.Request, params openapi.GetUsersParams) {
	search, ok := helper.ParseOptionalSearchQuery(w, r, params.Q, maxSearchQueryLen, h.OnError, "GetUsers", "Q")
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.user.UserUC.ListUsers(r.Context(), search, defaultUserSortField, page, perPage)
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

	if !h.requireOwnUserOrAdmin(w, r, user, userIDParsed, "GetUsersIDSolves", "AccessCheck") {
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

	if !h.requireOwnUserOrAdmin(w, r, user, userIDParsed, "GetUsersIDFails", "AccessCheck") {
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

	if !h.requireOwnUserOrAdmin(w, r, user, userIDParsed, "GetUsersIDAwards", "AccessCheck") {
		return
	}

	awards, err := h.user.UserUC.GetUserAwards(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUsersIDAwards", "GetUserAwards") {
		return
	}

	httputil.RenderOK(w, r, response.FromAwardList(awards))
}

var allowedUserListFields = []string{"username", "ip"}

// (GET /admin/users).
func (h *Server) GetAdminUsers(w http.ResponseWriter, r *http.Request, params openapi.GetAdminUsersParams) {
	field := defaultUserSortField

	if params.Field != nil {
		parsed, ok := httputil.ParseEnumQuery(r, "field", allowedUserListFields)
		if !ok {
			h.OnError(w, r, apperr.NewValidationErrorf("field must be one of: username, ip"), "GetAdminUsers", "Field")

			return
		}

		field = string(parsed)
	}

	q, ok := helper.ParseOptionalSearchQuery(w, r, params.Q, maxSearchQueryLen, h.OnError, "GetAdminUsers", "Q")
	if !ok {
		return
	}

	if field == sortFieldIP && q != nil && net.ParseIP(*params.Q) == nil {
		h.OnError(w, r, apperr.NewValidationErrorf("invalid IP address"), "GetAdminUsers", "Q")

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

	challenges, err := h.challenge.ChallengeUC.GetMissingChallengesByUserID(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetAdminUsersIDMissingChallenges", "GetMissingChallengesByUserID") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallenges(challenges))
}
