package response

import (
	"strconv"
	"time"

	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromAppSettings(s *domain.Settings) openapi.AppSettingsResponse {
	updatedAt := s.UpdatedAt.Format(time.RFC3339)
	return openapi.AppSettingsResponse{
		AppName:                          httputil.Ptr(s.AppName),
		CorsOrigins:                      httputil.Ptr(s.CORSOrigins),
		CsvExportMaxRows:                 httputil.Ptr(s.CSVExportMaxRows),
		DefaultPerPage:                   httputil.Ptr(s.DefaultPerPage),
		FrontendURL:                      httputil.Ptr(s.FrontendURL),
		MaxPerPage:                       httputil.Ptr(s.MaxPerPage),
		MaxTeams:                         httputil.Ptr(s.MaxTeams),
		RateLimitForgotPasswordPerMinute: httputil.Ptr(s.RateLimitForgotPasswordPerMinute),
		RateLimitGeneralIPPerMinute:      httputil.Ptr(s.RateLimitGeneralIPPerMinute),
		RateLimitLoginPerMinute:          httputil.Ptr(s.RateLimitLoginPerMinute),
		RateLimitLogoutPerMinute:         httputil.Ptr(s.RateLimitLogoutPerMinute),
		RateLimitRefreshPerMinute:        httputil.Ptr(s.RateLimitRefreshPerMinute),
		RateLimitRegisterPerMinute:       httputil.Ptr(s.RateLimitRegisterPerMinute),
		RateLimitResetPasswordPerMinute:  httputil.Ptr(s.RateLimitResetPasswordPerMinute),
		RateLimitScoreboardPerMinute:     httputil.Ptr(s.RateLimitScoreboardPerMinute),
		RateLimitVerifyEmailPerMinute:    httputil.Ptr(s.RateLimitVerifyEmailPerMinute),
		RateLimitOauthCallbackPerMinute:  httputil.Ptr(s.RateLimitOAuthCallbackPerMinute),
		RateLimitOauthRedirectPerMinute:  httputil.Ptr(s.RateLimitOAuthRedirectPerMinute),
		RateLimitCommentPerMinute:        httputil.Ptr(s.RateLimitCommentPerMinute),
		RegistrationOpen:                 httputil.Ptr(s.RegistrationOpen),
		ResendEnabled:                    httputil.Ptr(s.ResendEnabled),
		ResendFromEmail:                  httputil.Ptr(s.ResendFromEmail),
		ResendFromName:                   httputil.Ptr(s.ResendFromName),
		ResetTTLHours:                    httputil.Ptr(s.ResetTTLHours),
		ScoreboardVisible:                httputil.Ptr(s.ScoreboardVisible),
		SubmitLimitDurationMin:           httputil.Ptr(s.SubmitLimitDurationMin),
		SubmitLimitPerUser:               httputil.Ptr(s.SubmitLimitPerUser),
		VerifyEmails:                     httputil.Ptr(s.VerifyEmails),
		VerifyTTLHours:                   httputil.Ptr(s.VerifyTTLHours),
		WriteupEnabled:                   httputil.Ptr(s.WriteupEnabled),
		OauthGithubEnabled:               httputil.Ptr(s.OAuthGithubEnabled),
		OauthGoogleEnabled:               httputil.Ptr(s.OAuthGoogleEnabled),
		UpdatedAt:                        httputil.Ptr(updatedAt),
	}
}

func FromConfig(c *domain.CompetitionParam) openapi.ConfigResponse {
	res := openapi.ConfigResponse{
		Key:       c.Key,
		Value:     c.Value,
		ValueType: string(c.ValueType),
	}
	if c.Category != "" {
		res.Category = httputil.Ptr(c.Category)
	}
	if c.Description != "" {
		res.Description = httputil.Ptr(c.Description)
	}
	if !c.UpdatedAt.IsZero() {
		res.UpdatedAt = httputil.Ptr(c.UpdatedAt)
	}
	return res
}

func FromConfigResponseList(items []*domain.CompetitionParam) []openapi.ConfigResponse {
	return lo.Map(items, func(item *domain.CompetitionParam, _ int) openapi.ConfigResponse { return FromConfig(item) })
}

func FromConfigList(items []*domain.CompetitionParam) []openapi.ConfigResponse {
	return lo.Map(items, func(item *domain.CompetitionParam, _ int) openapi.ConfigResponse { return FromConfig(item) })
}

func FromConfigListToPublicMap(items []*domain.CompetitionParam) openapi.ConfigPublicResponse {
	out := make(openapi.ConfigPublicResponse, len(items))
	for _, c := range items {
		var v interface{}
		switch c.ValueType {
		case domain.CompetitionParamTypeBool:
			v = c.Value == "true"
		case domain.CompetitionParamTypeInt:
			if n, err := strconv.Atoi(c.Value); err == nil {
				v = n
			} else {
				v = c.Value
			}
		case domain.CompetitionParamTypeString, domain.CompetitionParamTypeJSON:
			v = c.Value
		default:
			v = c.Value
		}
		out[c.Key] = v
	}
	return out
}
