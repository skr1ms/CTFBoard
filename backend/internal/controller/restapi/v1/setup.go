package v1

import (
	"crypto/subtle"
	"net/http"

	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/setup"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

const setupTokenHeader = "X-Setup-Token"

// SetupHandler handles the first-run setup wizard endpoints.
type SetupHandler struct {
	uc                  *setup.SetupUseCase
	logger              logkit.Logger
	validator           validator.Validator
	setupToken          string
	secureCookies       bool
	refreshCookieMaxAge int
}

// NewSetupHandler constructs a SetupHandler.
func NewSetupHandler(uc *setup.SetupUseCase, logger logkit.Logger, v validator.Validator, setupToken string, secureCookies bool, refreshCookieMaxAge int) *SetupHandler {
	return &SetupHandler{
		uc:                  uc,
		logger:              logger,
		validator:           v,
		setupToken:          setupToken,
		secureCookies:       secureCookies,
		refreshCookieMaxAge: refreshCookieMaxAge,
	}
}

func (h *SetupHandler) validSetupToken(provided string) bool {
	if h.setupToken == "" || provided == "" || len(provided) != len(h.setupToken) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(provided), []byte(h.setupToken)) == 1
}

// GetSetupStatus returns whether the platform setup has been completed.
// This endpoint is public and always accessible.
func (h *SetupHandler) GetSetupStatus(w http.ResponseWriter, r *http.Request) {
	complete, err := h.uc.IsComplete(r.Context())
	if err != nil {
		h.onError(w, r, err)

		return
	}

	httputil.RenderOK(w, r, openapi.SetupStatusResponse{Complete: complete})
}

// PostSetup completes the first-run setup wizard.
// Returns 409 if setup has already been completed.
// On success returns the new admin's JWT access token and profile.
func (h *SetupHandler) PostSetup(w http.ResponseWriter, r *http.Request) {
	complete, err := h.uc.IsComplete(r.Context())
	if err != nil {
		h.onError(w, r, err)

		return
	}

	if complete {
		h.onError(w, r, apperr.ErrSetupAlreadyComplete)

		return
	}

	if !h.validSetupToken(r.Header.Get(setupTokenHeader)) {
		h.onError(w, r, apperr.ErrAccessDenied)

		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.SetupRequest](w, r, h.validator)
	if !ok {
		return
	}

	clientIP := kitMiddleware.GetClientIPFromContext(r.Context())

	result, err := h.uc.Complete(r.Context(), request.SetupRequestToParams(&req, clientIP))
	if err != nil {
		h.onError(w, r, err)

		return
	}

	refreshMaxAge := h.refreshCookieMaxAge
	if refreshMaxAge <= 0 {
		refreshMaxAge = defaultRefreshCookieMaxAge
	}

	setRefreshCookie(w, result.TokenPair.RefreshToken, refreshMaxAge, h.secureCookies)
	httputil.RenderOK(w, r, response.FromSetupComplete(result.TokenPair.AccessToken, result.User))
}

func (h *SetupHandler) onError(w http.ResponseWriter, r *http.Request, err error) {
	handler := &httputil.ErrorHandler{Logger: h.logger}
	handler.Handle(w, r, errmap.MapAppError(err), "SetupHandler")
}
