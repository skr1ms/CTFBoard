package request

import "github.com/skr1ms/CTFBoard/internal/entity"

type UpdateAppSettingsRequest struct {
	AppName                string `json:"app_name" validate:"required,max=100"`
	VerifyEmails           bool   `json:"verify_emails"`
	FrontendURL            string `json:"frontend_url" validate:"required,max=512"`
	CORSOrigins            string `json:"cors_origins" validate:"required"`
	ResendEnabled          bool   `json:"resend_enabled"`
	ResendFromEmail        string `json:"resend_from_email" validate:"required,max=255"`
	ResendFromName         string `json:"resend_from_name" validate:"required,max=100"`
	VerifyTTLHours         int    `json:"verify_ttl_hours" validate:"min=1,max=168"`
	ResetTTLHours          int    `json:"reset_ttl_hours" validate:"min=1,max=168"`
	SubmitLimitPerUser     int    `json:"submit_limit_per_user" validate:"min=1"`
	SubmitLimitDurationMin int    `json:"submit_limit_duration_min" validate:"min=1"`
	ScoreboardVisible      string `json:"scoreboard_visible" validate:"oneof=public hidden admins_only"`
	RegistrationOpen       bool   `json:"registration_open"`
}

func (r *UpdateAppSettingsRequest) ToAppSettings(ID int) *entity.AppSettings {
	return &entity.AppSettings{
		ID:                     ID,
		AppName:                r.AppName,
		VerifyEmails:           r.VerifyEmails,
		FrontendURL:            r.FrontendURL,
		CORSOrigins:            r.CORSOrigins,
		ResendEnabled:          r.ResendEnabled,
		ResendFromEmail:        r.ResendFromEmail,
		ResendFromName:         r.ResendFromName,
		VerifyTTLHours:         r.VerifyTTLHours,
		ResetTTLHours:          r.ResetTTLHours,
		SubmitLimitPerUser:     r.SubmitLimitPerUser,
		SubmitLimitDurationMin: r.SubmitLimitDurationMin,
		ScoreboardVisible:      r.ScoreboardVisible,
		RegistrationOpen:       r.RegistrationOpen,
	}
}
