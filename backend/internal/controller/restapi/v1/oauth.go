package v1

import (
	"net/http"
	"net/url"
	"slices"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const (
	oauthStateCookie       = "oauth_state"
	oauthStateCookieMaxAge = 600
)

// GetAuthOauthProvider redirects the user to the OAuth provider's authorization page.
// (GET /auth/oauth/{provider}).
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
		errMsg := sanitizeOAuthError(r.URL.Query().Get("error"))
		h.OnError(w, r, httperr.NewValidationErrorf("OAuth error: %s", errMsg), "GetAuthOauthProviderCallback", "MissingCode")

		return
	}

	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil || cookie.Value == "" {
		h.OnError(w, r, httperr.ErrOAuthStateMissing, "GetAuthOauthProviderCallback", "StateCookie")

		return
	}

	if !h.user.OAuthUC.ValidateState(cookie.Value, queryState) {
		h.OnError(w, r, httperr.ErrOAuthStateMismatch, "GetAuthOauthProviderCallback", "StateValidate")

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

var oauthErrorAllowlist = []string{
	"access_denied", "invalid_request", "unauthorized_client",
	"unsupported_response_type", "invalid_scope", "server_error", "temporarily_unavailable",
}

func sanitizeOAuthError(raw string) string {
	if slices.Contains(oauthErrorAllowlist, raw) {
		return raw
	}

	return "authorization code missing"
}
