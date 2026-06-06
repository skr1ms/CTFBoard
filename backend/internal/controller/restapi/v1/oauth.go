package v1

import (
	"net/http"
	"net/url"
	"slices"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	oauthStateCookie       = "oauth_state"
	oauthStateCookieMaxAge = 600
)

// oauthRedirectError clears the state cookie and redirects the browser to the
// frontend callback page with an ?error=<code> query parameter so the SPA can
// display a meaningful message instead of receiving a JSON error body.
func (h *Server) oauthRedirectError(w http.ResponseWriter, r *http.Request, code string) {
	h.clearOAuthStateCookie(w)

	q := url.Values{}
	q.Set("error", code)
	helper.RedirectFound(w, r, helper.FrontendCallbackURL(h.user.FrontendURL, q))
}

// oauthCodeFromErr extracts the application error code via errmap so that the
// OAuth redirect can carry a typed code the frontend understands.
func oauthCodeFromErr(err error) string {
	return errmap.Code(err)
}

// GetAuthOauthProvider redirects the user to the OAuth provider's authorization page
// (GET /auth/oauth/{provider}).
func (h *Server) GetAuthOauthProvider(w http.ResponseWriter, r *http.Request, provider string) {
	authURL, state, err := h.user.OAuthUC.GetAuthURL(r.Context(), provider)
	if h.OnError(w, r, err, "GetAuthOauthProvider", "GetAuthURL") {
		return
	}

	helper.SetHTTPOnlyCookie(w, helper.CookieOptions{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   oauthStateCookieMaxAge,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.user.SecureCookies,
	})

	helper.RedirectFound(w, r, authURL)
}

// GetAuthOauthProviderCallback handles the OAuth provider callback, exchanges
// the authorization code for tokens, and redirects with a short-lived exchange code.
// (GET /auth/oauth/{provider}/callback).
func (h *Server) GetAuthOauthProviderCallback(w http.ResponseWriter, r *http.Request, provider string, params openapi.GetAuthOauthProviderCallbackParams) {
	code := ""

	if params.Code != nil {
		code = *params.Code
	}

	queryState := ""

	if params.State != nil {
		queryState = *params.State
	}

	if code == "" {
		// The OAuth provider sent an error instead of a code. Pass the sanitized
		// RFC 6749 error code to the frontend (e.g. "access_denied").
		errCode := sanitizeOAuthError(r.URL.Query().Get("error"))
		h.oauthRedirectError(w, r, errCode)

		return
	}

	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil || cookie.Value == "" {
		h.oauthRedirectError(w, r, "OAUTH_STATE_MISSING")

		return
	}

	if !h.user.OAuthUC.ValidateState(cookie.Value, queryState) {
		h.oauthRedirectError(w, r, "OAUTH_STATE_MISMATCH")

		return
	}

	h.clearOAuthStateCookie(w)

	tokenPair, err := h.user.OAuthUC.HandleCallback(r.Context(), provider, code)
	if err != nil {
		h.oauthRedirectError(w, r, oauthCodeFromErr(err))

		return
	}

	exchangeCode, err := h.user.OAuthUC.IssueExchangeCode(r.Context(), tokenPair)
	if err != nil {
		h.oauthRedirectError(w, r, "INTERNAL_ERROR")

		return
	}

	q := url.Values{}
	q.Set("code", exchangeCode)
	helper.RedirectFound(w, r, helper.FrontendCallbackURL(h.user.FrontendURL, q))
}

func (h *Server) clearOAuthStateCookie(w http.ResponseWriter) {
	helper.ClearHTTPOnlyCookie(w, oauthStateCookie, "/", h.user.SecureCookies, http.SameSiteLaxMode)
}

var oauthErrorAllowlist = []string{
	"access_denied", "invalid_request", "unauthorized_client",
	"unsupported_response_type", "invalid_scope", "server_error", "temporarily_unavailable",
}

// sanitizeOAuthError returns raw only when it is one of the RFC 6749 standard
// error codes. Any unrecognized value is replaced with a generic message to
// prevent reflected content injection via the OAuth error query parameter.
func sanitizeOAuthError(raw string) string {
	if slices.Contains(oauthErrorAllowlist, raw) {
		return raw
	}

	return "authorization code missing"
}

// GetAuthOauthProviders returns which OAuth providers are configured on the server.
// GET /auth/oauth/providers - public, no auth required.
func (h *Server) GetAuthOauthProviders(w http.ResponseWriter, r *http.Request) {
	githubEnabled := h.user.OAuthGitHubEnabled
	googleEnabled := h.user.OAuthGoogleEnabled

	settings, err := h.admin.SettingsUC.Get(r.Context())
	if err != nil {
		h.infra.Logger.WithError(err).Warn("restapi - v1 - GetAuthOauthProviders - GetSettings failed, using config fallback")
	} else {
		githubEnabled = githubEnabled && settings.OAuthGithubEnabled
		googleEnabled = googleEnabled && settings.OAuthGoogleEnabled
	}

	httputil.RenderOK(w, r, response.FromOAuthProviders(githubEnabled, googleEnabled))
}

// PostAuthOauthExchange exchanges a one-time code (issued by the OAuth callback)
// for tokens. The code is consumed atomically - replay attempts return 404.
// POST /auth/oauth/exchange.
func (h *Server) PostAuthOauthExchange(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.OAuthExchangeRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	pair, err := h.user.OAuthUC.ConsumeExchangeCode(r.Context(), req.Code)
	if h.OnError(w, r, err, "PostAuthOauthExchange", "ConsumeExchangeCode") {
		return
	}

	h.setRefreshCookie(w, pair.RefreshToken)
	httputil.RenderOK(w, r, response.FromTokenPair(pair))
}
