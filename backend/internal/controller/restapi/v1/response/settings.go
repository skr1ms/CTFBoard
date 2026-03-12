package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromAppSettings(s *entity.Settings) openapi.AppSettingsResponse {
	updatedAt := s.UpdatedAt.Format(time.RFC3339)
	return openapi.AppSettingsResponse{
		AppName:                          ptr(s.AppName),
		CorsOrigins:                      ptr(s.CORSOrigins),
		CsvExportMaxRows:                 ptr(s.CSVExportMaxRows),
		DefaultPerPage:                   ptr(s.DefaultPerPage),
		FrontendURL:                      ptr(s.FrontendURL),
		MaxPerPage:                       ptr(s.MaxPerPage),
		MaxTeams:                         ptr(s.MaxTeams),
		RateLimitForgotPasswordPerMinute: ptr(s.RateLimitForgotPasswordPerMinute),
		RateLimitGeneralIPPerMinute:      ptr(s.RateLimitGeneralIPPerMinute),
		RateLimitLoginPerMinute:          ptr(s.RateLimitLoginPerMinute),
		RateLimitLogoutPerMinute:         ptr(s.RateLimitLogoutPerMinute),
		RateLimitRefreshPerMinute:        ptr(s.RateLimitRefreshPerMinute),
		RateLimitRegisterPerMinute:       ptr(s.RateLimitRegisterPerMinute),
		RateLimitResetPasswordPerMinute:  ptr(s.RateLimitResetPasswordPerMinute),
		RateLimitScoreboardPerMinute:     ptr(s.RateLimitScoreboardPerMinute),
		RateLimitVerifyEmailPerMinute:    ptr(s.RateLimitVerifyEmailPerMinute),
		RateLimitOauthCallbackPerMinute:  ptr(s.RateLimitOAuthCallbackPerMinute),
		RateLimitOauthRedirectPerMinute:  ptr(s.RateLimitOAuthRedirectPerMinute),
		RateLimitCommentPerMinute:        ptr(s.RateLimitCommentPerMinute),
		RegistrationOpen:                 ptr(s.RegistrationOpen),
		ResendEnabled:                    ptr(s.ResendEnabled),
		ResendFromEmail:                  ptr(s.ResendFromEmail),
		ResendFromName:                   ptr(s.ResendFromName),
		ResetTTLHours:                    ptr(s.ResetTTLHours),
		ScoreboardVisible:                ptr(s.ScoreboardVisible),
		SubmitLimitDurationMin:           ptr(s.SubmitLimitDurationMin),
		SubmitLimitPerUser:               ptr(s.SubmitLimitPerUser),
		UpdatedAt:                        &updatedAt,
		VerifyEmails:                     ptr(s.VerifyEmails),
		VerifyTTLHours:                   ptr(s.VerifyTTLHours),
		WriteupEnabled:                   ptr(s.WriteupEnabled),
		OauthGithubEnabled:               ptr(s.OAuthGithubEnabled),
		OauthGoogleEnabled:               ptr(s.OAuthGoogleEnabled),
	}
}

func FromConfig(c *entity.CompetitionParam) openapi.ConfigResponse {
	res := openapi.ConfigResponse{
		Key:       c.Key,
		Value:     c.Value,
		ValueType: string(c.ValueType),
	}
	if c.Description != "" {
		res.Description = ptr(c.Description)
	}
	if !c.UpdatedAt.IsZero() {
		res.UpdatedAt = ptr(c.UpdatedAt)
	}
	return res
}

func FromConfigList(items []*entity.CompetitionParam) []openapi.ConfigResponse {
	res := make([]openapi.ConfigResponse, len(items))
	for i, c := range items {
		res[i] = FromConfig(c)
	}
	return res
}
