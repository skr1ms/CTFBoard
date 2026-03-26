package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// List my API tokens
// (GET /user/tokens).
func (h *Server) GetUserTokens(w http.ResponseWriter, r *http.Request) {
	userIDParsed, ok := httputil.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	tokens, err := h.user.APITokenUC.List(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUserTokens", "List") {
		return
	}

	httputil.RenderOK(w, r, response.FromAPITokenList(tokens))
}

// Create API token
// (POST /user/tokens).
func (h *Server) PostUserTokens(w http.ResponseWriter, r *http.Request) {
	userIDParsed, ok := httputil.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.CreateAPITokenRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	description, expiresAt, err := request.CreateAPITokenRequestToParams(&req)
	if h.OnError(w, r, err, "PostUserTokens", "CreateAPITokenRequestToParams") {
		return
	}

	plaintext, token, err := h.user.APITokenUC.Create(r.Context(), userIDParsed, description, expiresAt)
	if h.OnError(w, r, err, "PostUserTokens", "Create") {
		return
	}

	httputil.RenderCreated(w, r, response.FromAPITokenCreated(plaintext, token))
}

// Revoke API token
// (DELETE /user/tokens/{ID}).
func (h *Server) DeleteUserTokensID(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := httputil.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	tokenIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.user.APITokenUC.Delete(r.Context(), tokenIDParsed, userIDParsed), "DeleteUserTokensID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}
