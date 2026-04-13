package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /user/tokens).
func (h *Server) GetUserTokens(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	tokens, err := h.user.APITokenUC.List(r.Context(), user.ID)
	if h.OnError(w, r, err, "GetUserTokens", "List") {
		return
	}

	httputil.RenderOK(w, r, response.FromAPITokenList(tokens))
}

// (POST /user/tokens).
func (h *Server) PostUserTokens(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
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

	plaintext, token, err := h.user.APITokenUC.Create(r.Context(), user.ID, description, expiresAt)
	if h.OnError(w, r, err, "PostUserTokens", "Create") {
		return
	}

	httputil.RenderCreated(w, r, response.FromAPITokenCreated(plaintext, token))
}

// (DELETE /user/tokens/{ID}).
func (h *Server) DeleteUserTokensID(w http.ResponseWriter, r *http.Request, ID string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	tokenIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.user.APITokenUC.Delete(r.Context(), tokenIDParsed, user.ID), "DeleteUserTokensID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}
