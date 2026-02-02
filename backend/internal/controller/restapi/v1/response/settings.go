package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromAppSettings(s *entity.AppSettings) openapi.ResponseAppSettingsResponse {
	appName := s.AppName
	corsOrigins := s.CORSOrigins
	frontendURL := s.FrontendURL
	resendFromEmail := s.ResendFromEmail
	resendFromName := s.ResendFromName
	scoreboardVisible := s.ScoreboardVisible
	updatedAt := s.UpdatedAt.Format(time.RFC3339)
	return openapi.ResponseAppSettingsResponse{
		AppName:                &appName,
		CorsOrigins:            &corsOrigins,
		FrontendURL:            &frontendURL,
		RegistrationOpen:       &s.RegistrationOpen,
		ResendEnabled:          &s.ResendEnabled,
		ResendFromEmail:        &resendFromEmail,
		ResendFromName:         &resendFromName,
		ResetTTLHours:          &s.ResetTTLHours,
		ScoreboardVisible:      &scoreboardVisible,
		SubmitLimitDurationMin: &s.SubmitLimitDurationMin,
		SubmitLimitPerUser:     &s.SubmitLimitPerUser,
		UpdatedAt:              &updatedAt,
		VerifyEmails:           &s.VerifyEmails,
		VerifyTTLHours:         &s.VerifyTTLHours,
	}
}
