package entity

import "time"

const (
	ScoreboardVisiblePublic     = "public"
	ScoreboardVisibleHidden     = "hidden"
	ScoreboardVisibleAdminsOnly = "admins_only"
)

type AppSettings struct {
	ID                     int       `json:"id"`
	AppName                string    `json:"app_name"`
	VerifyEmails           bool      `json:"verify_emails"`
	FrontendURL            string    `json:"frontend_url"`
	CORSOrigins            string    `json:"cors_origins"`
	ResendEnabled          bool      `json:"resend_enabled"`
	ResendFromEmail        string    `json:"resend_from_email"`
	ResendFromName         string    `json:"resend_from_name"`
	VerifyTTLHours         int       `json:"verify_ttl_hours"`
	ResetTTLHours          int       `json:"reset_ttl_hours"`
	SubmitLimitPerUser     int       `json:"submit_limit_per_user"`
	SubmitLimitDurationMin int       `json:"submit_limit_duration_min"`
	ScoreboardVisible      string    `json:"scoreboard_visible"`
	RegistrationOpen       bool      `json:"registration_open"`
	UpdatedAt              time.Time `json:"updated_at"`
}
