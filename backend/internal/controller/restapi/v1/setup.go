package v1

import (
	"encoding/json"
	"net/http"
	"time"

	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/setup"
)

// SetupHandler handles the first-run setup wizard endpoints.
type SetupHandler struct {
	uc     *setup.SetupUseCase
	logger logkit.Logger
}

// NewSetupHandler constructs a SetupHandler.
func NewSetupHandler(uc *setup.SetupUseCase, logger logkit.Logger) *SetupHandler {
	return &SetupHandler{uc: uc, logger: logger}
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

	return ""
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

	httputil.RenderOK(w, r, map[string]any{
		"token": result.TokenPair.AccessToken,
		"user":  response.FromUser(result.User),
	})
}

func (h *SetupHandler) onError(w http.ResponseWriter, r *http.Request, err error) {
	handler := &httputil.ErrorHandler{Logger: h.logger}
	handler.Handle(w, r, errmap.MapAppError(err), "SetupHandler")
}
