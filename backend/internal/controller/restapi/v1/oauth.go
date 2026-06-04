package v1

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-jwtkit"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const (
	oauthStateCookie       = "oauth_state"
	oauthStateCookieMaxAge = 600

	oauthExchangePrefix = "oauth_exchange:"
	oauthExchangeTTL    = 30 * time.Second
	oauthExchangeBytes  = 32
)

type storedTokenPair struct {
	AccessToken      string `json:"a"`
	RefreshToken     string `json:"r"`
	AccessExpiresAt  int64  `json:"ae"`
	RefreshExpiresAt int64  `json:"re"`
}

func generateExchangeCode() (string, error) {
	b := make([]byte, oauthExchangeBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func (h *Server) storeExchangeCode(ctx context.Context, code string, pair *jwtkit.TokenPair) error {
	val, err := json.Marshal(storedTokenPair{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
	})
	if err != nil {
		return err
	}

	return h.infra.RedisClient.Set(ctx, oauthExchangePrefix+code, val, oauthExchangeTTL).Err()
}

func (h *Server) consumeExchangeCode(ctx context.Context, code string) (*jwtkit.TokenPair, error) {
	val, err := h.infra.RedisClient.GetDel(ctx, oauthExchangePrefix+code).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, apperr.ErrTokenNotFound
		}

		return nil, err
	}

	var stored storedTokenPair
	if err := json.Unmarshal(val, &stored); err != nil {
		return nil, err
	}

	return &jwtkit.TokenPair{
		AccessToken:      stored.AccessToken,
		RefreshToken:     stored.RefreshToken,
		AccessExpiresAt:  stored.AccessExpiresAt,
		RefreshExpiresAt: stored.RefreshExpiresAt,
	}, nil
}

// oauthRedirectError clears the state cookie and redirects the browser to the
// frontend callback page with an ?error=<code> query parameter so the SPA can
// display a meaningful message instead of receiving a JSON error body.
func (h *Server) oauthRedirectError(w http.ResponseWriter, r *http.Request, code string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.user.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	q := url.Values{}
	q.Set("error", code)
	http.Redirect(w, r, h.user.FrontendURL+"/auth/callback?"+q.Encode(), http.StatusFound)
}

// oauthCodeFromErr extracts the application error code via errmap so that the
// OAuth redirect can carry a typed code the frontend understands.
func oauthCodeFromErr(err error) string {
	mapped := errmap.MapAppError(err)

	var he *httperr.HTTPError
	if errors.As(mapped, &he) {
		return he.GetCode()
	}

	return "INTERNAL_ERROR"
}

// GetAuthOauthProvider redirects the user to the OAuth provider's authorization page
// (GET /auth/oauth/{provider}).
func (h *Server) GetAuthOauthProvider(w http.ResponseWriter, r *http.Request, provider string) {
	authURL, state, err := h.user.OAuthUC.GetAuthURL(r.Context(), provider)
	if h.OnError(w, r, err, "GetAuthOauthProvider", "GetAuthURL") {
		return
	}

	http.SetCookie(w, &http.Cookie{ // nosemgrep: go.lang.security.audit.net.cookie-missing-secure.cookie-missing-secure
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
// authorization code for tokens and redirects to the frontend with tokens in the fragment
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

	http.SetCookie(w, &http.Cookie{ // nosemgrep: go.lang.security.audit.net.cookie-missing-secure.cookie-missing-secure
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.user.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	tokenPair, err := h.user.OAuthUC.HandleCallback(r.Context(), provider, code)
	if err != nil {
		h.oauthRedirectError(w, r, oauthCodeFromErr(err))

		return
	}

	// Generate a short-lived one-time code so tokens never appear in the URL.
	exchangeCode, err := generateExchangeCode()
	if err != nil {
		h.oauthRedirectError(w, r, "INTERNAL_ERROR")

		return
	}

	if err := h.storeExchangeCode(r.Context(), exchangeCode, tokenPair); err != nil {
		h.oauthRedirectError(w, r, "INTERNAL_ERROR")

		return
	}

	q := url.Values{}
	q.Set("code", exchangeCode)
	http.Redirect(w, r, h.user.FrontendURL+"/auth/callback?"+q.Encode(), http.StatusFound)
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

	httputil.RenderOK(w, r, map[string]bool{
		"github": githubEnabled,
		"google": googleEnabled,
	})
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

	pair, err := h.consumeExchangeCode(r.Context(), req.Code)
	if h.OnError(w, r, err, "PostAuthOauthExchange", "consumeExchangeCode") {
		return
	}

	h.setRefreshCookie(w, pair.RefreshToken)
	httputil.RenderOK(w, r, response.FromTokenPair(pair))
}
