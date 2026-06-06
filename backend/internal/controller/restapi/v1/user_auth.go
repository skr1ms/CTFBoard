package v1

import (
	"net/http"
	"strings"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	refreshCookieName          = "ctf_refresh"
	defaultRefreshCookieMaxAge = 72 * 60 * 60
)

func setRefreshCookie(w http.ResponseWriter, token string, maxAge int, secure bool) {
	helper.SetHTTPOnlyCookie(w, helper.CookieOptions{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		MaxAge:   maxAge,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Server) setRefreshCookie(w http.ResponseWriter, token string) {
	maxAge := h.user.RefreshCookieMaxAge
	if maxAge <= 0 {
		maxAge = defaultRefreshCookieMaxAge
	}

	setRefreshCookie(w, token, maxAge, h.user.SecureCookies)
}

func (h *Server) clearRefreshCookie(w http.ResponseWriter) {
	setRefreshCookie(w, "", -1, h.user.SecureCookies)
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

	h.setRefreshCookie(w, tokenPair.RefreshToken)
	httputil.RenderOK(w, r, response.FromTokenValues(tokenPair.AccessToken, tokenPair.AccessExpiresAt))
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
// (POST /auth/refresh).
func (h *Server) PostAuthRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		h.OnError(w, r, helper.ErrNotAuthenticated, "PostAuthRefresh", "MissingRefreshCookie")

		return
	}

	tokenPair, err := h.user.UserUC.RefreshTokenPair(r.Context(), cookie.Value)
	if h.OnError(w, r, err, "PostAuthRefresh", "RefreshTokenPair") {
		return
	}

	h.setRefreshCookie(w, tokenPair.RefreshToken)
	httputil.RenderOK(w, r, response.FromTokenValues(tokenPair.AccessToken, tokenPair.AccessExpiresAt))
}

// PostAuthLogout revokes the refresh token from the httpOnly cookie and
// optionally revokes the current access token from the Authorization header.
// (POST /auth/logout).
func (h *Server) PostAuthLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		h.OnError(w, r, helper.ErrNotAuthenticated, "PostAuthLogout", "MissingRefreshCookie")

		return
	}

	var accessToken *string

	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if token, ok := strings.CutPrefix(authHeader, "Bearer "); ok && token != "" {
			accessToken = &token
		}
	}

	if err := h.user.UserUC.Logout(r.Context(), cookie.Value, accessToken); h.OnError(w, r, err, "PostAuthLogout", "Logout") {
		return
	}

	h.clearRefreshCookie(w)
	httputil.RenderNoContent(w, r)
}
