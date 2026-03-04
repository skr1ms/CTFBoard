package v1

import (
	"net/http"
	"net/url"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	oauthStateCookie       = "oauth_state"
	oauthStateCookieMaxAge = 600
)

// GetAuthOauthProvider redirects the user to the OAuth provider's authorization page.
// (GET /auth/oauth/{provider})
func (h *Server) GetAuthOauthProvider(w http.ResponseWriter, r *http.Request, provider string) {
	authURL, state, err := h.user.OAuthUC.GetAuthURL(r.Context(), provider)
	if h.OnError(w, r, err, "GetAuthOauthProvider", "GetAuthURL") {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   oauthStateCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.user.SecureCookies,
	})

	http.Redirect(w, r, authURL, http.StatusFound)
}

// GetAuthOauthProviderCallback handles the OAuth provider callback, exchanges the
// authorization code for tokens and redirects to the frontend with tokens in the fragment.
// (GET /auth/oauth/{provider}/callback)
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
		errMsg := sanitizeOAuthError(r.URL.Query().Get("error"))
		h.OnError(w, r, helper.NewValidationErrorf("OAuth error: %s", errMsg), "GetAuthOauthProviderCallback", "MissingCode")
		return
	}

	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil || cookie.Value == "" {
		h.OnError(w, r, helper.ErrOAuthStateMissing, "GetAuthOauthProviderCallback", "StateCookie")
		return
	}

	if !h.user.OAuthUC.ValidateState(cookie.Value, queryState) {
		h.OnError(w, r, helper.ErrOAuthStateMismatch, "GetAuthOauthProviderCallback", "StateValidate")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.user.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	tokenPair, err := h.user.OAuthUC.HandleCallback(r.Context(), provider, code)
	if h.OnError(w, r, err, "GetAuthOauthProviderCallback", "HandleCallback") {
		return
	}

	redirectURL, err := url.Parse(h.user.FrontendURL + "/auth/callback")
	if h.OnError(w, r, err, "GetAuthOauthProviderCallback", "ParseFrontendURL") {
		return
	}

	fragment := url.Values{}
	fragment.Set("access_token", tokenPair.AccessToken)
	fragment.Set("refresh_token", tokenPair.RefreshToken)
	redirectURL.Fragment = fragment.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// oauthErrorAllowlist contains RFC 6749 standard error codes returned by OAuth providers.
var oauthErrorAllowlist = map[string]bool{
	"access_denied":             true,
	"invalid_request":           true,
	"unauthorized_client":       true,
	"unsupported_response_type": true,
	"invalid_scope":             true,
	"server_error":              true,
	"temporarily_unavailable":   true,
}

// sanitizeOAuthError returns the error code only if it is a known OAuth RFC 6749 value,
// otherwise returns a generic message to prevent log injection via user-supplied query params.
func sanitizeOAuthError(raw string) string {
	if oauthErrorAllowlist[raw] {
		return raw
	}
	return "authorization code missing"
}
