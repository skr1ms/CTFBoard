package entity

import "time"

const (
	ScoreboardVisiblePublic     = "public"
	ScoreboardVisibleHidden     = "hidden"
	ScoreboardVisibleAdminsOnly = "admins_only"
)

type Settings struct {
	ID                               int       `json:"id"`
	AppName                          string    `json:"app_name"`
	VerifyEmails                     bool      `json:"verify_emails"`
	FrontendURL                      string    `json:"frontend_url"`
	CORSOrigins                      string    `json:"cors_origins"`
	ResendEnabled                    bool      `json:"resend_enabled"`
	ResendFromEmail                  string    `json:"resend_from_email"`
	ResendFromName                   string    `json:"resend_from_name"`
	VerifyTTLHours                   int       `json:"verify_ttl_hours"`
	ResetTTLHours                    int       `json:"reset_ttl_hours"`
	SubmitLimitPerUser               int       `json:"submit_limit_per_user"`
	SubmitLimitDurationMin           int       `json:"submit_limit_duration_min"`
	ScoreboardVisible                string    `json:"scoreboard_visible"`
	RegistrationOpen                 bool      `json:"registration_open"`
	DefaultPerPage                   int       `json:"default_per_page"`
	MaxPerPage                       int       `json:"max_per_page"`
	CSVExportMaxRows                 int       `json:"csv_export_max_rows"`
	RateLimitLoginPerMinute          int       `json:"rate_limit_login_per_minute"`
	RateLimitRegisterPerMinute       int       `json:"rate_limit_register_per_minute"`
	RateLimitForgotPasswordPerMinute int       `json:"rate_limit_forgot_password_per_minute"`
	RateLimitResetPasswordPerMinute  int       `json:"rate_limit_reset_password_per_minute"`
	RateLimitLogoutPerMinute         int       `json:"rate_limit_logout_per_minute"`
	RateLimitRefreshPerMinute        int       `json:"rate_limit_refresh_per_minute"`
	RateLimitScoreboardPerMinute     int       `json:"rate_limit_scoreboard_per_minute"`
	RateLimitGeneralIPPerMinute      int       `json:"rate_limit_general_ip_per_minute"`
	RateLimitVerifyEmailPerMinute    int       `json:"rate_limit_verify_email_per_minute"`
	RateLimitOAuthCallbackPerMinute  int       `json:"rate_limit_oauth_callback_per_minute"`
	MaxTeams                         int       `json:"max_teams"`
	WriteupEnabled                   bool      `json:"writeup_enabled"`
	OAuthGithubEnabled               bool      `json:"oauth_github_enabled"`
	OAuthGoogleEnabled               bool      `json:"oauth_google_enabled"`
	UpdatedAt                        time.Time `json:"updated_at"`
}
