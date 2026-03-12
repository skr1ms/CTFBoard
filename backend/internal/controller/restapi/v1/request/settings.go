package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxRateLimitPerMinute  = 10000
	minRateLimitPerMinute  = 1
	minPerPage             = 1
	maxPerPage             = 1000
	minCSVExportMaxRows    = 1
	maxAppNameLen          = 100
	maxFrontendURLLen      = 512
	maxCORSOriginsLen      = 2048
	maxResendFromEmailLen  = 255
	maxResendFromNameLen   = 100
	minTTLHours            = 1
	maxTTLHours            = 168
	minSubmitLimitPerUser  = 1
	minSubmitLimitDuration = 1
	minMaxTeams            = 0
)

type rateLimitField struct {
	name  string
	value *int
}

func ValidateUpdateAppSettingsRequest(req *openapi.UpdateAppSettingsRequest) error {
	fields := []rateLimitField{
		{"rate_limit_login_per_minute", req.RateLimitLoginPerMinute},
		{"rate_limit_register_per_minute", req.RateLimitRegisterPerMinute},
		{"rate_limit_forgot_password_per_minute", req.RateLimitForgotPasswordPerMinute},
		{"rate_limit_reset_password_per_minute", req.RateLimitResetPasswordPerMinute},
		{"rate_limit_logout_per_minute", req.RateLimitLogoutPerMinute},
		{"rate_limit_refresh_per_minute", req.RateLimitRefreshPerMinute},
		{"rate_limit_scoreboard_per_minute", req.RateLimitScoreboardPerMinute},
		{"rate_limit_general_ip_per_minute", req.RateLimitGeneralIPPerMinute},
		{"rate_limit_verify_email_per_minute", req.RateLimitVerifyEmailPerMinute},
		{"rate_limit_oauth_callback_per_minute", req.RateLimitOauthCallbackPerMinute},
		{"rate_limit_oauth_redirect_per_minute", req.RateLimitOauthRedirectPerMinute},
		{"rate_limit_comment_per_minute", req.RateLimitCommentPerMinute},
	}
	for _, f := range fields {
		if f.value != nil {
			if *f.value < minRateLimitPerMinute {
				return helper.NewValidationErrorf("%s must be at least %d", f.name, minRateLimitPerMinute)
			}
			if *f.value > maxRateLimitPerMinute {
				return helper.NewValidationErrorf("%s must not exceed %d", f.name, maxRateLimitPerMinute)
			}
		}
	}
	if req.DefaultPerPage != nil && (*req.DefaultPerPage < minPerPage || *req.DefaultPerPage > maxPerPage) {
		return helper.NewValidationErrorf("default_per_page must be between %d and %d", minPerPage, maxPerPage)
	}
	if req.MaxPerPage != nil && (*req.MaxPerPage < minPerPage || *req.MaxPerPage > maxPerPage) {
		return helper.NewValidationErrorf("max_per_page must be between %d and %d", minPerPage, maxPerPage)
	}
	if req.CsvExportMaxRows != nil && *req.CsvExportMaxRows < minCSVExportMaxRows {
		return helper.NewValidationErrorf("csv_export_max_rows must be at least %d", minCSVExportMaxRows)
	}
	if req.MaxTeams != nil && *req.MaxTeams < minMaxTeams {
		return helper.NewValidationErrorf("max_teams must be at least %d", minMaxTeams)
	}
	if req.SubmitLimitPerUser != nil && *req.SubmitLimitPerUser < minSubmitLimitPerUser {
		return helper.NewValidationErrorf("submit_limit_per_user must be at least %d", minSubmitLimitPerUser)
	}
	if req.SubmitLimitDurationMin != nil && *req.SubmitLimitDurationMin < minSubmitLimitDuration {
		return helper.NewValidationErrorf("submit_limit_duration_min must be at least %d", minSubmitLimitDuration)
	}
	if req.VerifyTTLHours != nil && (*req.VerifyTTLHours < minTTLHours || *req.VerifyTTLHours > maxTTLHours) {
		return helper.NewValidationErrorf("verify_ttl_hours must be between %d and %d", minTTLHours, maxTTLHours)
	}
	if req.ResetTTLHours != nil && (*req.ResetTTLHours < minTTLHours || *req.ResetTTLHours > maxTTLHours) {
		return helper.NewValidationErrorf("reset_ttl_hours must be between %d and %d", minTTLHours, maxTTLHours)
	}
	if req.AppName != nil && len(*req.AppName) > maxAppNameLen {
		return helper.NewValidationErrorf("app_name must be at most %d characters", maxAppNameLen)
	}
	if req.FrontendURL != nil && len(*req.FrontendURL) > maxFrontendURLLen {
		return helper.NewValidationErrorf("frontend_url must be at most %d characters", maxFrontendURLLen)
	}
	if req.CorsOrigins != nil && len(*req.CorsOrigins) > maxCORSOriginsLen {
		return helper.NewValidationErrorf("cors_origins must be at most %d characters", maxCORSOriginsLen)
	}
	if req.ResendFromEmail != nil && len(*req.ResendFromEmail) > maxResendFromEmailLen {
		return helper.NewValidationErrorf("resend_from_email must be at most %d characters", maxResendFromEmailLen)
	}
	if req.ResendFromName != nil && len(*req.ResendFromName) > maxResendFromNameLen {
		return helper.NewValidationErrorf("resend_from_name must be at most %d characters", maxResendFromNameLen)
	}
	return nil
}

func UpdateAppSettingsRequestToEntity(req *openapi.UpdateAppSettingsRequest, id int, current *entity.Settings) *entity.Settings {
	scoreboardVisible := current.ScoreboardVisible
	if req.ScoreboardVisible != nil {
		scoreboardVisible = string(*req.ScoreboardVisible)
	}
	registrationOpen := current.RegistrationOpen
	if req.RegistrationOpen != nil {
		registrationOpen = *req.RegistrationOpen
	}
	return &entity.Settings{
		ID:                               id,
		AppName:                          derefOr(req.AppName, current.AppName),
		VerifyEmails:                     derefOr(req.VerifyEmails, current.VerifyEmails),
		FrontendURL:                      derefOr(req.FrontendURL, current.FrontendURL),
		CORSOrigins:                      derefOr(req.CorsOrigins, current.CORSOrigins),
		ResendEnabled:                    derefOr(req.ResendEnabled, current.ResendEnabled),
		ResendFromEmail:                  derefOr(req.ResendFromEmail, current.ResendFromEmail),
		ResendFromName:                   derefOr(req.ResendFromName, current.ResendFromName),
		VerifyTTLHours:                   derefOr(req.VerifyTTLHours, current.VerifyTTLHours),
		ResetTTLHours:                    derefOr(req.ResetTTLHours, current.ResetTTLHours),
		SubmitLimitPerUser:               derefOr(req.SubmitLimitPerUser, current.SubmitLimitPerUser),
		SubmitLimitDurationMin:           derefOr(req.SubmitLimitDurationMin, current.SubmitLimitDurationMin),
		ScoreboardVisible:                scoreboardVisible,
		RegistrationOpen:                 registrationOpen,
		DefaultPerPage:                   derefOr(req.DefaultPerPage, current.DefaultPerPage),
		MaxPerPage:                       derefOr(req.MaxPerPage, current.MaxPerPage),
		CSVExportMaxRows:                 derefOr(req.CsvExportMaxRows, current.CSVExportMaxRows),
		RateLimitLoginPerMinute:          derefOr(req.RateLimitLoginPerMinute, current.RateLimitLoginPerMinute),
		RateLimitRegisterPerMinute:       derefOr(req.RateLimitRegisterPerMinute, current.RateLimitRegisterPerMinute),
		RateLimitForgotPasswordPerMinute: derefOr(req.RateLimitForgotPasswordPerMinute, current.RateLimitForgotPasswordPerMinute),
		RateLimitResetPasswordPerMinute:  derefOr(req.RateLimitResetPasswordPerMinute, current.RateLimitResetPasswordPerMinute),
		RateLimitLogoutPerMinute:         derefOr(req.RateLimitLogoutPerMinute, current.RateLimitLogoutPerMinute),
		RateLimitRefreshPerMinute:        derefOr(req.RateLimitRefreshPerMinute, current.RateLimitRefreshPerMinute),
		RateLimitScoreboardPerMinute:     derefOr(req.RateLimitScoreboardPerMinute, current.RateLimitScoreboardPerMinute),
		RateLimitGeneralIPPerMinute:      derefOr(req.RateLimitGeneralIPPerMinute, current.RateLimitGeneralIPPerMinute),
		RateLimitVerifyEmailPerMinute:    derefOr(req.RateLimitVerifyEmailPerMinute, current.RateLimitVerifyEmailPerMinute),
		RateLimitOAuthCallbackPerMinute:  derefOr(req.RateLimitOauthCallbackPerMinute, current.RateLimitOAuthCallbackPerMinute),
		RateLimitOAuthRedirectPerMinute:  derefOr(req.RateLimitOauthRedirectPerMinute, current.RateLimitOAuthRedirectPerMinute),
		RateLimitCommentPerMinute:        derefOr(req.RateLimitCommentPerMinute, current.RateLimitCommentPerMinute),
		MaxTeams:                         derefOr(req.MaxTeams, current.MaxTeams),
		WriteupEnabled:                   derefOr(req.WriteupEnabled, current.WriteupEnabled),
		OAuthGithubEnabled:               derefOr(req.OauthGithubEnabled, current.OAuthGithubEnabled),
		OAuthGoogleEnabled:               derefOr(req.OauthGoogleEnabled, current.OAuthGoogleEnabled),
	}
}
