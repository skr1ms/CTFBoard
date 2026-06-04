package v1

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/setup"
)

const setupTokenHeader = "X-Setup-Token"

// SetupHandler handles the first-run setup wizard endpoints.
type SetupHandler struct {
	uc                  *setup.SetupUseCase
	logger              logkit.Logger
	setupToken          string
	secureCookies       bool
	refreshCookieMaxAge int
}

// NewSetupHandler constructs a SetupHandler.
func NewSetupHandler(uc *setup.SetupUseCase, logger logkit.Logger, setupToken string, secureCookies bool, refreshCookieMaxAge int) *SetupHandler {
	return &SetupHandler{
		uc:                  uc,
		logger:              logger,
		setupToken:          setupToken,
		secureCookies:       secureCookies,
		refreshCookieMaxAge: refreshCookieMaxAge,
	}
}

type setupStatusResponse struct {
	Complete bool `json:"complete"`
}

type setupRequest struct {
	CTFName                   string     `json:"ctf_name"`
	CTFDescription            string     `json:"ctf_description"`
	Mode                      string     `json:"mode"`
	MaxTeamSize               int        `json:"max_team_size"`
	ChallengeVisibility       string     `json:"challenge_visibility"`
	ScoreVisibility           string     `json:"score_visibility"`
	AccountVisibility         string     `json:"account_visibility"`
	RegistrationVisibility    string     `json:"registration_visibility"`
	EmailVerificationRequired bool       `json:"email_verification_required"`
	AdminUsername             string     `json:"admin_username"`
	AdminEmail                string     `json:"admin_email"`
	AdminPassword             string     `json:"admin_password"`
	StartTime                 *time.Time `json:"start_time,omitempty"`
	EndTime                   *time.Time `json:"end_time,omitempty"`
	FreezeTime                *time.Time `json:"freeze_time,omitempty"`
	Timezone                  string     `json:"timezone"`
}

func (req *setupRequest) validate() string {
	switch {
	case req.CTFName == "":
		return "ctf_name is required"
	case req.AdminUsername == "":
		return "admin_username is required"
	case req.AdminEmail == "":
		return "admin_email is required"
	case len(req.AdminPassword) < 8:
		return "admin_password must be at least 8 characters"
	case req.Mode == "" || (req.Mode != "teams_only" && req.Mode != "solo_only" && req.Mode != "flexible"):
		return "mode must be one of: teams_only, solo_only, flexible"
	}

	if msg := validateVisibilityValue("challenge_visibility", req.ChallengeVisibility, []string{"public", "private", "hidden", "admins"}); msg != "" {
		return msg
	}

	if msg := validateVisibilityValue("score_visibility", req.ScoreVisibility, []string{"public", "private", "hidden", "admins"}); msg != "" {
		return msg
	}

	if msg := validateVisibilityValue("account_visibility", req.AccountVisibility, []string{"public", "private", "hidden", "admins"}); msg != "" {
		return msg
	}

	if msg := validateVisibilityValue("registration_visibility", req.RegistrationVisibility, []string{"public", "private"}); msg != "" {
		return msg
	}

	return ""
}

func validateVisibilityValue(key, value string, allowed []string) string {
	if !slices.Contains(allowed, value) {
		return key + " must be one of: " + strings.Join(allowed, ", ")
	}

	return ""
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

	httputil.RenderOK(w, r, setupStatusResponse{Complete: complete})
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

	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"INVALID_JSON","message":"invalid request body"}`, http.StatusBadRequest)

		return
	}

	if msg := req.validate(); msg != "" {
		http.Error(w, `{"error":"VALIDATION_ERROR","message":"`+msg+`"}`, http.StatusBadRequest)

		return
	}

	clientIP := kitMiddleware.GetClientIPFromContext(r.Context())

	result, err := h.uc.Complete(r.Context(), &setup.SetupRequest{
		CTFName:                   req.CTFName,
		CTFDescription:            req.CTFDescription,
		Mode:                      req.Mode,
		MaxTeamSize:               req.MaxTeamSize,
		ChallengeVisibility:       req.ChallengeVisibility,
		ScoreVisibility:           req.ScoreVisibility,
		AccountVisibility:         req.AccountVisibility,
		RegistrationVisibility:    req.RegistrationVisibility,
		EmailVerificationRequired: req.EmailVerificationRequired,
		AdminUsername:             req.AdminUsername,
		AdminEmail:                req.AdminEmail,
		AdminPassword:             req.AdminPassword,
		StartTime:                 req.StartTime,
		EndTime:                   req.EndTime,
		FreezeTime:                req.FreezeTime,
		Timezone:                  req.Timezone,
		ClientIP:                  clientIP,
	})
	if err != nil {
		h.onError(w, r, err)

		return
	}

	refreshMaxAge := h.refreshCookieMaxAge
	if refreshMaxAge <= 0 {
		refreshMaxAge = defaultRefreshCookieMaxAge
	}

	setRefreshCookie(w, result.TokenPair.RefreshToken, refreshMaxAge, h.secureCookies)
	httputil.RenderOK(w, r, map[string]any{
		"token": result.TokenPair.AccessToken,
		"user":  response.FromUser(result.User),
	})
}

func (h *SetupHandler) onError(w http.ResponseWriter, r *http.Request, err error) {
	handler := &httputil.ErrorHandler{Logger: h.logger}
	handler.Handle(w, r, errmap.MapAppError(err), "SetupHandler")
}
