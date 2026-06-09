package response

import (
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromAppSettings(s *domain.Settings) openapi.AppSettingsResponse {
	return openapi.AppSettingsResponse{
		AppName:                          new(s.AppName),
		CorsOrigins:                      new(s.CORSOrigins),
		CsvExportMaxRows:                 new(s.CSVExportMaxRows),
		DefaultPerPage:                   new(s.DefaultPerPage),
		FrontendURL:                      new(s.FrontendURL),
		MaxPerPage:                       new(s.MaxPerPage),
		MaxUsers:                         new(s.MaxUsers),
		MaxTeams:                         new(s.MaxTeams),
		RateLimitForgotPasswordPerMinute: new(s.RateLimitForgotPasswordPerMinute),
		RateLimitGeneralIPPerMinute:      new(s.RateLimitGeneralIPPerMinute),
		RateLimitLoginPerMinute:          new(s.RateLimitLoginPerMinute),
		RateLimitLogoutPerMinute:         new(s.RateLimitLogoutPerMinute),
		RateLimitRefreshPerMinute:        new(s.RateLimitRefreshPerMinute),
		RateLimitRegisterPerMinute:       new(s.RateLimitRegisterPerMinute),
		RateLimitResetPasswordPerMinute:  new(s.RateLimitResetPasswordPerMinute),
		RateLimitScoreboardPerMinute:     new(s.RateLimitScoreboardPerMinute),
		RateLimitVerifyEmailPerMinute:    new(s.RateLimitVerifyEmailPerMinute),
		RateLimitOauthCallbackPerMinute:  new(s.RateLimitOAuthCallbackPerMinute),
		RateLimitOauthRedirectPerMinute:  new(s.RateLimitOAuthRedirectPerMinute),
		RateLimitCommentPerMinute:        new(s.RateLimitCommentPerMinute),
		RegistrationOpen:                 new(s.RegistrationOpen),
		ResendEnabled:                    new(s.ResendEnabled),
		ResendFromEmail:                  new(s.ResendFromEmail),
		ResendFromName:                   new(s.ResendFromName),
		ResetTTLHours:                    new(s.ResetTTLHours),
		SubmitLimitDurationMin:           new(s.SubmitLimitDurationMin),
		SubmitLimitPerUser:               new(s.SubmitLimitPerUser),
		VerifyEmails:                     new(s.VerifyEmails),
		VerifyTTLHours:                   new(s.VerifyTTLHours),
		WriteupEnabled:                   new(s.WriteupEnabled),
		OauthGithubEnabled:               new(s.OAuthGithubEnabled),
		OauthGoogleEnabled:               new(s.OAuthGoogleEnabled),
		UpdatedAt:                        timePtr(&s.UpdatedAt),
	}
}

func FromConfig(c *domain.CompetitionParam) openapi.ConfigResponse {
	res := openapi.ConfigResponse{
		Key:       c.Key,
		Value:     c.Value,
		ValueType: string(c.ValueType),
	}
	if c.Category != "" {
		res.Category = new(c.Category)
	}

	if c.Description != "" {
		res.Description = new(c.Description)
	}

	if !c.UpdatedAt.IsZero() {
		res.UpdatedAt = new(c.UpdatedAt)
	}

	return res
}

func FromConfigCategories(items []*domain.CompetitionParam) []openapi.ConfigCategoryItem {
	counts := make(map[string]int)

	for _, p := range items {
		if p.Category != "" {
			counts[p.Category]++
		}
	}

	out := make([]openapi.ConfigCategoryItem, 0, len(counts))
	for name, count := range counts {
		out = append(out, openapi.ConfigCategoryItem{Name: name, Count: count})
	}

	slices.SortFunc(out, func(a, b openapi.ConfigCategoryItem) int { return strings.Compare(a.Name, b.Name) })

	return out
}

func FromConfigResponseList(items []*domain.CompetitionParam) []openapi.ConfigResponse {
	return lo.Map(items, func(item *domain.CompetitionParam, _ int) openapi.ConfigResponse { return FromConfig(item) })
}

func FromConfigListToPublicMap(items []*domain.CompetitionParam) openapi.ConfigPublicResponse {
	out := make(openapi.ConfigPublicResponse, len(items))
	for _, c := range items {
		var v any

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
