package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// User login
// (POST /auth/login)
func (h *Server) PostAuthLogin(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestLoginRequest](
		w, r, h.validator, h.logger, "PostAuthLogin",
	)
	if !ok {
		return
	}

	email, password := request.LoginRequestCredentials(&req)
	tokenPair, err := h.userUC.Login(r.Context(), email, password)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthLogin")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromTokenPair(tokenPair))
}

// Register new user
// (POST /auth/register)
func (h *Server) PostAuthRegister(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestRegisterRequest](
		w, r, h.validator, h.logger, "PostAuthRegister",
	)
	if !ok {
		return
	}

	username, email, password := request.RegisterRequestCredentials(&req)
	user, err := h.userUC.Register(r.Context(), username, email, password)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthRegister")
		handleError(w, r, err)
		return
	}

	if err := h.emailUC.SendVerificationEmail(r.Context(), user); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthRegister - SendVerificationEmail")
	}

	RenderCreated(w, r, response.FromUserForRegister(user))
}

// Get current user info
// (GET /auth/me)
func (h *Server) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUser(r.Context())
	if !ok {
		RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	RenderOK(w, r, response.FromUserForMe(user))
}

// Get user profile
// (GET /users/{ID})
func (h *Server) GetUsersID(w http.ResponseWriter, r *http.Request, ID string) {
	useruuid, err := uuid.Parse(ID)
	if err != nil {
		RenderInvalidID(w, r)
		return
	}

	profile, err := h.userUC.GetProfile(r.Context(), useruuid)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetUsersID")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromUserProfile(profile))
}
