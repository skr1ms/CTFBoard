package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// User login
// (POST /auth/login)
func (h *Server) PostAuthLogin(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RequestLoginRequest](
		w, r, h.validator, h.logger, "PostAuthLogin",
	)
	if !ok {
		return
	}

	email, password := request.LoginRequestCredentials(&req)
	tokenPair, err := h.userUC.Login(r.Context(), email, password)
	if h.OnError(w, r, err, "PostAuthLogin", "Login") {
		return
	}

	helper.RenderOK(w, r, response.FromTokenPair(tokenPair))
}

// Register new user
// (POST /auth/register)
func (h *Server) PostAuthRegister(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RequestRegisterRequest](
		w, r, h.validator, h.logger, "PostAuthRegister",
	)
	if !ok {
		return
	}

	username, email, password, customFields := request.RegisterRequestCredentials(&req)
	user, err := h.userUC.Register(r.Context(), username, email, password, customFields)
	if h.OnError(w, r, err, "PostAuthRegister", "Register") {
		return
	}

	if err := h.emailUC.SendVerificationEmail(r.Context(), user); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthRegister - SendVerificationEmail")
	}

	helper.RenderCreated(w, r, response.FromUserForRegister(user))
}

// Get current user info
// (GET /auth/me)
func (h *Server) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	helper.RenderOK(w, r, response.FromUserForMe(user))
}

// Get user profile
// (GET /users/{ID})
func (h *Server) GetUsersID(w http.ResponseWriter, r *http.Request, ID string) {
	useruuid, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	profile, err := h.userUC.GetProfile(r.Context(), useruuid)
	if h.OnError(w, r, err, "GetUsersID", "GetProfile") {
		return
	}

	helper.RenderOK(w, r, response.FromUserProfile(profile))
}
