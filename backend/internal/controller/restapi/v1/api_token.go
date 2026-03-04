package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// List my API tokens
// (GET /user/tokens)
func (h *Server) GetUserTokens(w http.ResponseWriter, r *http.Request) {
	userIDParsed, ok := helper.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	tokens, err := h.user.APITokenUC.List(r.Context(), userIDParsed)
	if h.OnError(w, r, err, "GetUserTokens", "List") {
		return
	}
	helper.RenderOK(w, r, response.FromAPITokenList(tokens))
}

// Create API token
// (POST /user/tokens)
func (h *Server) PostUserTokens(w http.ResponseWriter, r *http.Request) {
	userIDParsed, ok := helper.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.CreateAPITokenRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostUserTokens",
	)
	if !ok {
		return
	}

	description, expiresAt := request.CreateAPITokenRequestToParams(&req)
	plaintext, token, err := h.user.APITokenUC.Create(r.Context(), userIDParsed, description, expiresAt)
	if h.OnError(w, r, err, "PostUserTokens", "Create") {
		return
	}

	helper.RenderCreated(w, r, response.FromAPITokenCreated(plaintext, token))
}

// Revoke API token
// (DELETE /user/tokens/{ID})
func (h *Server) DeleteUserTokensID(w http.ResponseWriter, r *http.Request, ID string) {
	userIDParsed, ok := helper.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	tokenIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.user.APITokenUC.Delete(r.Context(), tokenIDParsed, userIDParsed), "DeleteUserTokensID", "Delete") {
		return
	}

	helper.RenderNoContent(w, r)
}
