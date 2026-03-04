package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const maxRateLimitPerMinute = 10000

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
	}
	for _, f := range fields {
		if f.value != nil && *f.value > maxRateLimitPerMinute {
			return helper.NewValidationErrorf("%s must not exceed %d", f.name, maxRateLimitPerMinute)
		}
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
		AppName:                          req.AppName,
		VerifyEmails:                     derefOr(req.VerifyEmails, current.VerifyEmails),
		FrontendURL:                      req.FrontendURL,
		CORSOrigins:                      req.CorsOrigins,
		ResendEnabled:                    derefOr(req.ResendEnabled, current.ResendEnabled),
		ResendFromEmail:                  req.ResendFromEmail,
		ResendFromName:                   req.ResendFromName,
		VerifyTTLHours:                   derefOr(req.VerifyTTLHours, current.VerifyTTLHours),
		ResetTTLHours:                    derefOr(req.ResetTTLHours, current.ResetTTLHours),
		SubmitLimitPerUser:               derefOr(req.SubmitLimitPerUser, current.SubmitLimitPerUser),
		SubmitLimitDurationMin:           derefOr(req.SubmitLimitDurationMin, current.SubmitLimitDurationMin),
		ScoreboardVisible:                scoreboardVisible,
		RegistrationOpen:                 registrationOpen,
		DefaultPerPage:                   derefOr(req.DefaultPerPage, current.DefaultPerPage),
		MaxPerPage:                       derefOr(req.MaxPerPage, current.MaxPerPage),
		CSVExportMaxRows:                 derefOr(req.CsvExportMaxRows, current.CSVExportMaxRows),
		RateLimitLoginPerMinute:          derefOr(req.RateLimitLoginPerMinute, 0),
		RateLimitRegisterPerMinute:       derefOr(req.RateLimitRegisterPerMinute, 0),
		RateLimitForgotPasswordPerMinute: derefOr(req.RateLimitForgotPasswordPerMinute, 0),
		RateLimitResetPasswordPerMinute:  derefOr(req.RateLimitResetPasswordPerMinute, 0),
		RateLimitLogoutPerMinute:         derefOr(req.RateLimitLogoutPerMinute, 0),
		RateLimitRefreshPerMinute:        derefOr(req.RateLimitRefreshPerMinute, 0),
		RateLimitScoreboardPerMinute:     derefOr(req.RateLimitScoreboardPerMinute, 0),
		RateLimitGeneralIPPerMinute:      derefOr(req.RateLimitGeneralIPPerMinute, 0),
		RateLimitVerifyEmailPerMinute:    derefOr(req.RateLimitVerifyEmailPerMinute, 0),
		RateLimitOAuthCallbackPerMinute:  derefOr(req.RateLimitOauthCallbackPerMinute, 0),
		MaxTeams:                         derefOr(req.MaxTeams, current.MaxTeams),
		WriteupEnabled:                   derefOr(req.WriteupEnabled, current.WriteupEnabled),
		OAuthGithubEnabled:               derefOr(req.OauthGithubEnabled, current.OAuthGithubEnabled),
		OAuthGoogleEnabled:               derefOr(req.OauthGoogleEnabled, current.OAuthGoogleEnabled),
	}
}
