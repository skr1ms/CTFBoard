package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// List my API tokens
// (GET /user/tokens)
func (h *Server) GetUserTokens(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	tokens, err := h.apiTokenUC.List(r.Context(), userID)
	if h.OnError(w, r, err, "GetUserTokens", "List") {
		return
	}

	res := make([]openapi.ResponseAPITokenResponse, len(tokens))
	for i, t := range tokens {
		res[i] = response.FromAPIToken(t)
	}
	helper.RenderOK(w, r, res)
}

// Create API token
// (POST /user/tokens)
func (h *Server) PostUserTokens(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.RequestCreateAPITokenRequest](
		w, r, h.validator, h.logger, "PostUserTokens",
	)
	if !ok {
		return
	}

	description, expiresAt := request.CreateAPITokenParams(&req)
	plaintext, token, err := h.apiTokenUC.Create(r.Context(), userID, description, expiresAt)
	if h.OnError(w, r, err, "PostUserTokens", "Create") {
		return
	}

	helper.RenderCreated(w, r, response.FromAPITokenCreated(plaintext, token))
}

// Revoke API token
// (DELETE /user/tokens/{ID})
func (h *Server) DeleteUserTokensID(w http.ResponseWriter, r *http.Request, id string) {
	userID, ok := helper.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	tokenID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}

	if h.OnError(w, r, h.apiTokenUC.Delete(r.Context(), tokenID, userID), "DeleteUserTokensID", "Delete") {
		return
	}

	helper.RenderNoContent(w, r)
}
