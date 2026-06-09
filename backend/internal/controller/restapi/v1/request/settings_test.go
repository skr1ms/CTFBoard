package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestUpdateAppSettingsRequestToEntityAllowsDeployTimeFieldsWhenUnchanged(t *testing.T) {
	t.Parallel()

	current := settingsRequestCurrent()
	appName := "new app"
	frontendURL := current.FrontendURL
	corsOrigins := current.CORSOrigins

	got, err := UpdateAppSettingsRequestToEntity(&openapi.UpdateAppSettingsRequest{
		AppName:     &appName,
		FrontendURL: &frontendURL,
		CorsOrigins: &corsOrigins,
	}, current.ID, current)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, appName, got.AppName)
	assert.Equal(t, current.FrontendURL, got.FrontendURL)
	assert.Equal(t, current.CORSOrigins, got.CORSOrigins)
}

func TestUpdateAppSettingsRequestToEntityRejectsDeployTimeFieldChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*openapi.UpdateAppSettingsRequest)
		want   string
	}{
		{
			name: "frontend url",
			mutate: func(req *openapi.UpdateAppSettingsRequest) {
				value := "https://new.example.com"
				req.FrontendURL = &value
			},
			want: "frontend_url is deploy-time only",
		},
		{
			name: "cors origins",
			mutate: func(req *openapi.UpdateAppSettingsRequest) {
				value := "https://new.example.com"
				req.CorsOrigins = &value
			},
			want: "cors_origins is deploy-time only",
		},
		{
			name: "resend enabled",
			mutate: func(req *openapi.UpdateAppSettingsRequest) {
				value := true
				req.ResendEnabled = &value
			},
			want: "resend_enabled is deploy-time only",
		},
		{
			name: "resend from email",
			mutate: func(req *openapi.UpdateAppSettingsRequest) {
				value := "ops@example.com"
				req.ResendFromEmail = &value
			},
			want: "resend_from_email is deploy-time only",
		},
		{
			name: "resend from name",
			mutate: func(req *openapi.UpdateAppSettingsRequest) {
				value := "Ops"
				req.ResendFromName = &value
			},
			want: "resend_from_name is deploy-time only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := &openapi.UpdateAppSettingsRequest{}
			tt.mutate(req)

			got, err := UpdateAppSettingsRequestToEntity(req, 1, settingsRequestCurrent())

			require.Error(t, err)
			assert.Nil(t, got)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func settingsRequestCurrent() *domain.Settings {
	return &domain.Settings{
		ID:                               1,
		AppName:                          "old app",
		FrontendURL:                      "https://app.example.com",
		CORSOrigins:                      "https://app.example.com",
		SubmitLimitPerUser:               1,
		SubmitLimitDurationMin:           1,
		VerifyTTLHours:                   24,
		ResetTTLHours:                    2,
		DefaultPerPage:                   20,
		MaxPerPage:                       100,
		CSVExportMaxRows:                 1000,
		RateLimitLoginPerMinute:          5,
		RateLimitRegisterPerMinute:       5,
		RateLimitForgotPasswordPerMinute: 5,
		RateLimitResetPasswordPerMinute:  5,
		RateLimitLogoutPerMinute:         5,
		RateLimitRefreshPerMinute:        5,
		RateLimitScoreboardPerMinute:     5,
		RateLimitGeneralIPPerMinute:      60,
		RateLimitVerifyEmailPerMinute:    5,
		RateLimitOAuthCallbackPerMinute:  5,
		RateLimitOAuthRedirectPerMinute:  5,
		RateLimitCommentPerMinute:        5,
		ResendEnabled:                    true,
		ResendFromEmail:                  "noreply@example.com",
		ResendFromName:                   "Astro CTF",
	}
}
