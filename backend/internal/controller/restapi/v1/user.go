package v1

import (
	"net/http"

	"github.com/google/uuid"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

// User login
// (POST /auth/login)
func (h *Server) PostAuthLogin(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.LoginRequest](
		w, r, h.validator, h.logger, "PostAuthLogin",
	)
	if !ok {
		return
	}

	tokenPair, err := h.userUC.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthLogin")
		handleError(w, r, err)
		return
	}

	res := openapi.JwtTokenPair{
		AccessToken:      ptr(tokenPair.AccessToken),
		AccessExpiresAt:  ptr(int(tokenPair.AccessExpiresAt)),
		RefreshToken:     ptr(tokenPair.RefreshToken),
		RefreshExpiresAt: ptr(int(tokenPair.RefreshExpiresAt)),
	}

	httputil.RenderOK(w, r, res)
}

// Register new user
// (POST /auth/register)
func (h *Server) PostAuthRegister(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.RegisterRequest](
		w, r, h.validator, h.logger, "PostAuthRegister",
	)
	if !ok {
		return
	}

	user, err := h.userUC.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthRegister")
		handleError(w, r, err)
		return
	}

	if err := h.emailUC.SendVerificationEmail(r.Context(), user); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthRegister - SendVerificationEmail")
	}

	res := response.FromUserForRegister(user)

	httputil.RenderCreated(w, r, res)
}

// Get current user info
// (GET /auth/me)
func (h *Server) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	res := response.FromUserForMe(user)

	httputil.RenderOK(w, r, res)
}

// Get user profile
// (GET /users/{ID})
func (h *Server) GetUsersID(w http.ResponseWriter, r *http.Request, ID string) {
	useruuid, err := uuid.Parse(ID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	profile, err := h.userUC.GetProfile(r.Context(), useruuid)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetUsersID")
		handleError(w, r, err)
		return
	}

	res := response.FromUserProfile(profile)

	httputil.RenderOK(w, r, res)
}
