package persistent

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
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
		RateLimitOAuthRedirectPerMinute:  int(s.RateLimitOAuthRedirectPerMinute),
		RateLimitCommentPerMinute:        int(s.RateLimitCommentPerMinute),
		MaxTeams:                         int(s.MaxTeams),
		WriteupEnabled:                   s.WriteupEnabled,
		OAuthGithubEnabled:               s.OAuthGithubEnabled,
		OAuthGoogleEnabled:               s.OAuthGoogleEnabled,
		UpdatedAt:                        ptrTimeToTime(timestamptzToTime(s.UpdatedAt)),
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

func (r *SettingsRepo) GetForUpdate(ctx context.Context) (*entity.Settings, error) {
	s, err := r.q(ctx).GetAppSettingsForUpdate(ctx)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrAppSettingsNotFound
		}
		return nil, fmt.Errorf("SettingsRepo - GetForUpdate: %w", err)
	}
	return toEntityAppSettings(s), nil
}

func (r *SettingsRepo) Update(ctx context.Context, s *entity.Settings) error {
	verifyTTL, err := intToInt32Safe(s.VerifyTTLHours)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - VerifyTTLHours: %w", err)
	}
	resetTTL, err := intToInt32Safe(s.ResetTTLHours)
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
	rateLimitOAuthRedirect, err := intToInt32Safe(s.RateLimitOAuthRedirectPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitOAuthRedirectPerMinute: %w", err)
	}
	rateLimitComment, err := intToInt32Safe(s.RateLimitCommentPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update - RateLimitCommentPerMinute: %w", err)
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
		VerifyTTLHours:                   verifyTTL,
		ResetTTLHours:                    resetTTL,
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
		RateLimitOAuthRedirectPerMinute:  rateLimitOAuthRedirect,
		RateLimitCommentPerMinute:        rateLimitComment,
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

func (r *SettingsRepo) UpdateIfCurrent(ctx context.Context, s *entity.Settings) error {
	verifyTTL, err := intToInt32Safe(s.VerifyTTLHours)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - VerifyTTLHours: %w", err)
	}
	resetTTL, err := intToInt32Safe(s.ResetTTLHours)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - ResetTTLHours: %w", err)
	}
	submitLimit, err := intToInt32Safe(s.SubmitLimitPerUser)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - SubmitLimitPerUser: %w", err)
	}
	submitDuration, err := intToInt32Safe(s.SubmitLimitDurationMin)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - SubmitLimitDurationMin: %w", err)
	}
	defaultPerPage, err := intToInt32Safe(s.DefaultPerPage)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - DefaultPerPage: %w", err)
	}
	maxPerPage, err := intToInt32Safe(s.MaxPerPage)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - MaxPerPage: %w", err)
	}
	csvExportMaxRows, err := intToInt32Safe(s.CSVExportMaxRows)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - CSVExportMaxRows: %w", err)
	}
	rateLimitLogin, err := intToInt32Safe(s.RateLimitLoginPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitLoginPerMinute: %w", err)
	}
	rateLimitRegister, err := intToInt32Safe(s.RateLimitRegisterPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitRegisterPerMinute: %w", err)
	}
	rateLimitForgotPassword, err := intToInt32Safe(s.RateLimitForgotPasswordPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitForgotPasswordPerMinute: %w", err)
	}
	rateLimitResetPassword, err := intToInt32Safe(s.RateLimitResetPasswordPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitResetPasswordPerMinute: %w", err)
	}
	rateLimitLogout, err := intToInt32Safe(s.RateLimitLogoutPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitLogoutPerMinute: %w", err)
	}
	rateLimitRefresh, err := intToInt32Safe(s.RateLimitRefreshPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitRefreshPerMinute: %w", err)
	}
	rateLimitScoreboard, err := intToInt32Safe(s.RateLimitScoreboardPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitScoreboardPerMinute: %w", err)
	}
	rateLimitGeneralIP, err := intToInt32Safe(s.RateLimitGeneralIPPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitGeneralIPPerMinute: %w", err)
	}
	rateLimitVerifyEmail, err := intToInt32Safe(s.RateLimitVerifyEmailPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitVerifyEmailPerMinute: %w", err)
	}
	rateLimitOAuthCallback, err := intToInt32Safe(s.RateLimitOAuthCallbackPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitOAuthCallbackPerMinute: %w", err)
	}
	rateLimitOAuthRedirect, err := intToInt32Safe(s.RateLimitOAuthRedirectPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitOAuthRedirectPerMinute: %w", err)
	}
	rateLimitComment, err := intToInt32Safe(s.RateLimitCommentPerMinute)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - RateLimitCommentPerMinute: %w", err)
	}
	maxTeams, err := intToInt32Safe(s.MaxTeams)
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent - MaxTeams: %w", err)
	}
	_, err = r.q(ctx).UpdateAppSettingsIfCurrent(ctx, sqlc.UpdateAppSettingsIfCurrentParams{
		AppName:                          s.AppName,
		VerifyEmails:                     s.VerifyEmails,
		FrontendURL:                      s.FrontendURL,
		CORSOrigins:                      s.CORSOrigins,
		ResendEnabled:                    s.ResendEnabled,
		ResendFromEmail:                  s.ResendFromEmail,
		ResendFromName:                   s.ResendFromName,
		VerifyTTLHours:                   verifyTTL,
		ResetTTLHours:                    resetTTL,
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
		RateLimitOAuthRedirectPerMinute:  rateLimitOAuthRedirect,
		RateLimitCommentPerMinute:        rateLimitComment,
		MaxTeams:                         maxTeams,
		WriteupEnabled:                   s.WriteupEnabled,
		OAuthGithubEnabled:               s.OAuthGithubEnabled,
		OAuthGoogleEnabled:               s.OAuthGoogleEnabled,
		UpdatedAt:                        timeToTimestamptz(&s.UpdatedAt),
	})
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrSettingsConflict
		}
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent: %w", err)
	}
	return nil
}
