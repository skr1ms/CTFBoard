package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func UpdateAppSettingsRequestToEntity(req *openapi.UpdateAppSettingsRequest, id int, current *domain.Settings) *domain.Settings {
	scoreboardVisible := current.ScoreboardVisible
	if req.ScoreboardVisible != nil {
		scoreboardVisible = string(*req.ScoreboardVisible)
	}

	registrationOpen := current.RegistrationOpen
	if req.RegistrationOpen != nil {
		registrationOpen = *req.RegistrationOpen
	}

	return &domain.Settings{
		ID:                               id,
		UpdatedAt:                        current.UpdatedAt,
		AppName:                          lo.FromPtrOr(req.AppName, current.AppName),
		VerifyEmails:                     lo.FromPtrOr(req.VerifyEmails, current.VerifyEmails),
		FrontendURL:                      lo.FromPtrOr(req.FrontendURL, current.FrontendURL),
		CORSOrigins:                      lo.FromPtrOr(req.CorsOrigins, current.CORSOrigins),
		ResendEnabled:                    lo.FromPtrOr(req.ResendEnabled, current.ResendEnabled),
		ResendFromEmail:                  lo.FromPtrOr(req.ResendFromEmail, current.ResendFromEmail),
		ResendFromName:                   lo.FromPtrOr(req.ResendFromName, current.ResendFromName),
		VerifyTTLHours:                   lo.FromPtrOr(req.VerifyTTLHours, current.VerifyTTLHours),
		ResetTTLHours:                    lo.FromPtrOr(req.ResetTTLHours, current.ResetTTLHours),
		SubmitLimitPerUser:               lo.FromPtrOr(req.SubmitLimitPerUser, current.SubmitLimitPerUser),
		SubmitLimitDurationMin:           lo.FromPtrOr(req.SubmitLimitDurationMin, current.SubmitLimitDurationMin),
		ScoreboardVisible:                scoreboardVisible,
		RegistrationOpen:                 registrationOpen,
		DefaultPerPage:                   lo.FromPtrOr(req.DefaultPerPage, current.DefaultPerPage),
		MaxPerPage:                       lo.FromPtrOr(req.MaxPerPage, current.MaxPerPage),
		CSVExportMaxRows:                 lo.FromPtrOr(req.CsvExportMaxRows, current.CSVExportMaxRows),
		RateLimitLoginPerMinute:          lo.FromPtrOr(req.RateLimitLoginPerMinute, current.RateLimitLoginPerMinute),
		RateLimitRegisterPerMinute:       lo.FromPtrOr(req.RateLimitRegisterPerMinute, current.RateLimitRegisterPerMinute),
		RateLimitForgotPasswordPerMinute: lo.FromPtrOr(req.RateLimitForgotPasswordPerMinute, current.RateLimitForgotPasswordPerMinute),
		RateLimitResetPasswordPerMinute:  lo.FromPtrOr(req.RateLimitResetPasswordPerMinute, current.RateLimitResetPasswordPerMinute),
		RateLimitLogoutPerMinute:         lo.FromPtrOr(req.RateLimitLogoutPerMinute, current.RateLimitLogoutPerMinute),
		RateLimitRefreshPerMinute:        lo.FromPtrOr(req.RateLimitRefreshPerMinute, current.RateLimitRefreshPerMinute),
		RateLimitScoreboardPerMinute:     lo.FromPtrOr(req.RateLimitScoreboardPerMinute, current.RateLimitScoreboardPerMinute),
		RateLimitGeneralIPPerMinute:      lo.FromPtrOr(req.RateLimitGeneralIPPerMinute, current.RateLimitGeneralIPPerMinute),
		RateLimitVerifyEmailPerMinute:    lo.FromPtrOr(req.RateLimitVerifyEmailPerMinute, current.RateLimitVerifyEmailPerMinute),
		RateLimitOAuthCallbackPerMinute:  lo.FromPtrOr(req.RateLimitOauthCallbackPerMinute, current.RateLimitOAuthCallbackPerMinute),
		RateLimitOAuthRedirectPerMinute:  lo.FromPtrOr(req.RateLimitOauthRedirectPerMinute, current.RateLimitOAuthRedirectPerMinute),
		RateLimitCommentPerMinute:        lo.FromPtrOr(req.RateLimitCommentPerMinute, current.RateLimitCommentPerMinute),
		MaxTeams:                         lo.FromPtrOr(req.MaxTeams, current.MaxTeams),
		WriteupEnabled:                   lo.FromPtrOr(req.WriteupEnabled, current.WriteupEnabled),
		OAuthGithubEnabled:               lo.FromPtrOr(req.OauthGithubEnabled, current.OAuthGithubEnabled),
		OAuthGoogleEnabled:               lo.FromPtrOr(req.OauthGoogleEnabled, current.OAuthGoogleEnabled),
	}
}
