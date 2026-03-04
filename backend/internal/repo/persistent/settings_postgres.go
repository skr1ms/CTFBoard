package persistent

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsRepo struct {
	pool *pgxpool.Pool
}

var _ repo.SettingsRepository = (*SettingsRepo)(nil)

func NewSettingsRepo(pool *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{pool: pool}
}

func (r *SettingsRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func toEntityAppSettings(s sqlc.AppSettings) *entity.Settings {
	return &entity.Settings{
		ID:                               int(s.ID),
		AppName:                          s.AppName,
		VerifyEmails:                     s.VerifyEmails,
		FrontendURL:                      s.FrontendURL,
		CORSOrigins:                      s.CORSOrigins,
		ResendEnabled:                    s.ResendEnabled,
		ResendFromEmail:                  s.ResendFromEmail,
		ResendFromName:                   s.ResendFromName,
		VerifyTTLHours:                   int(s.VerifyTTLHours),
		ResetTTLHours:                    int(s.ResetTTLHours),
		SubmitLimitPerUser:               int(s.SubmitLimitPerUser),
		SubmitLimitDurationMin:           int(s.SubmitLimitDurationMin),
		ScoreboardVisible:                s.ScoreboardVisible,
		RegistrationOpen:                 s.RegistrationOpen,
		DefaultPerPage:                   int(s.DefaultPerPage),
		MaxPerPage:                       int(s.MaxPerPage),
		CSVExportMaxRows:                 int(s.CSVExportMaxRows),
		RateLimitLoginPerMinute:          int(s.RateLimitLoginPerMinute),
		RateLimitRegisterPerMinute:       int(s.RateLimitRegisterPerMinute),
		RateLimitForgotPasswordPerMinute: int(s.RateLimitForgotPasswordPerMinute),
		RateLimitResetPasswordPerMinute:  int(s.RateLimitResetPasswordPerMinute),
		RateLimitLogoutPerMinute:         int(s.RateLimitLogoutPerMinute),
		RateLimitRefreshPerMinute:        int(s.RateLimitRefreshPerMinute),
		RateLimitScoreboardPerMinute:     int(s.RateLimitScoreboardPerMinute),
		RateLimitGeneralIPPerMinute:      int(s.RateLimitGeneralIPPerMinute),
		RateLimitVerifyEmailPerMinute:    int(s.RateLimitVerifyEmailPerMinute),
		RateLimitOAuthCallbackPerMinute:  int(s.RateLimitOAuthCallbackPerMinute),
		MaxTeams:                         int(s.MaxTeams),
		WriteupEnabled:                   s.WriteupEnabled,
		OAuthGithubEnabled:               s.OAuthGithubEnabled,
		OAuthGoogleEnabled:               s.OAuthGoogleEnabled,
		UpdatedAt:                        s.UpdatedAt,
	}
}

func (r *SettingsRepo) Get(ctx context.Context) (*entity.Settings, error) {
	s, err := r.q(ctx).GetAppSettings(ctx)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrAppSettingsNotFound
		}
		return nil, fmt.Errorf("SettingsRepo - Get: %w", err)
	}
	return toEntityAppSettings(s), nil
}

//nolint:gocognit,gocyclo,funlen
func (r *SettingsRepo) Update(ctx context.Context, s *entity.Settings) error {
	verifyTtl, err := intToInt32Safe(s.VerifyTTLHours)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - VerifyTTLHours: %w", err)
	}
	resetTtl, err := intToInt32Safe(s.ResetTTLHours)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - ResetTTLHours: %w", err)
	}
	submitLimit, err := intToInt32Safe(s.SubmitLimitPerUser)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - SubmitLimitPerUser: %w", err)
	}
	submitDuration, err := intToInt32Safe(s.SubmitLimitDurationMin)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - SubmitLimitDurationMin: %w", err)
	}
	defaultPerPage, err := intToInt32Safe(s.DefaultPerPage)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - DefaultPerPage: %w", err)
	}
	maxPerPage, err := intToInt32Safe(s.MaxPerPage)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - MaxPerPage: %w", err)
	}
	csvExportMaxRows, err := intToInt32Safe(s.CSVExportMaxRows)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - CSVExportMaxRows: %w", err)
	}
	rateLimitLogin, err := intToInt32Safe(s.RateLimitLoginPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitLoginPerMinute: %w", err)
	}
	rateLimitRegister, err := intToInt32Safe(s.RateLimitRegisterPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitRegisterPerMinute: %w", err)
	}
	rateLimitForgotPassword, err := intToInt32Safe(s.RateLimitForgotPasswordPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitForgotPasswordPerMinute: %w", err)
	}
	rateLimitResetPassword, err := intToInt32Safe(s.RateLimitResetPasswordPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitResetPasswordPerMinute: %w", err)
	}
	rateLimitLogout, err := intToInt32Safe(s.RateLimitLogoutPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitLogoutPerMinute: %w", err)
	}
	rateLimitRefresh, err := intToInt32Safe(s.RateLimitRefreshPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitRefreshPerMinute: %w", err)
	}
	rateLimitScoreboard, err := intToInt32Safe(s.RateLimitScoreboardPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitScoreboardPerMinute: %w", err)
	}
	rateLimitGeneralIP, err := intToInt32Safe(s.RateLimitGeneralIPPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitGeneralIPPerMinute: %w", err)
	}
	rateLimitVerifyEmail, err := intToInt32Safe(s.RateLimitVerifyEmailPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitVerifyEmailPerMinute: %w", err)
	}
	rateLimitOAuthCallback, err := intToInt32Safe(s.RateLimitOAuthCallbackPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitOAuthCallbackPerMinute: %w", err)
	}
	maxTeams, err := intToInt32Safe(s.MaxTeams)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - MaxTeams: %w", err)
	}
	err = r.q(ctx).UpdateAppSettings(ctx, sqlc.UpdateAppSettingsParams{
		AppName:                          s.AppName,
		VerifyEmails:                     s.VerifyEmails,
		FrontendURL:                      s.FrontendURL,
		CORSOrigins:                      s.CORSOrigins,
		ResendEnabled:                    s.ResendEnabled,
		ResendFromEmail:                  s.ResendFromEmail,
		ResendFromName:                   s.ResendFromName,
		VerifyTTLHours:                   verifyTtl,
		ResetTTLHours:                    resetTtl,
		SubmitLimitPerUser:               submitLimit,
		SubmitLimitDurationMin:           submitDuration,
		ScoreboardVisible:                s.ScoreboardVisible,
		RegistrationOpen:                 s.RegistrationOpen,
		DefaultPerPage:                   defaultPerPage,
		MaxPerPage:                       maxPerPage,
		CSVExportMaxRows:                 csvExportMaxRows,
		RateLimitLoginPerMinute:          rateLimitLogin,
		RateLimitRegisterPerMinute:       rateLimitRegister,
		RateLimitForgotPasswordPerMinute: rateLimitForgotPassword,
		RateLimitResetPasswordPerMinute:  rateLimitResetPassword,
		RateLimitLogoutPerMinute:         rateLimitLogout,
		RateLimitRefreshPerMinute:        rateLimitRefresh,
		RateLimitScoreboardPerMinute:     rateLimitScoreboard,
		RateLimitGeneralIPPerMinute:      rateLimitGeneralIP,
		RateLimitVerifyEmailPerMinute:    rateLimitVerifyEmail,
		RateLimitOAuthCallbackPerMinute:  rateLimitOAuthCallback,
		MaxTeams:                         maxTeams,
		WriteupEnabled:                   s.WriteupEnabled,
		OAuthGithubEnabled:               s.OAuthGithubEnabled,
		OAuthGoogleEnabled:               s.OAuthGoogleEnabled,
	})
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update: %w", err)
	}
	return nil
}
