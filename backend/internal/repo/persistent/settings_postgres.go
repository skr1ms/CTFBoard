package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type SettingsRepo struct {
	BaseRepo
}

var _ repo.SettingsRepository = (*SettingsRepo)(nil)

func NewSettingsRepo(pool *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{BaseRepo: BaseRepo{pool: pool}}
}

func toDomainAppSettings(s sqlc.AppSettings) *domain.Settings {
	return &domain.Settings{
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
		UpdatedAt:                        pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(s.UpdatedAt)),
	}
}

func (r *SettingsRepo) Get(ctx context.Context) (*domain.Settings, error) {
	s, err := GetOrNotFound(func() (sqlc.AppSettings, error) { return r.Q(ctx).GetAppSettings(ctx) },
		apperr.ErrAppSettingsNotFound, "SettingsRepo - Get")
	if err != nil {
		return nil, err
	}

	return toDomainAppSettings(s), nil
}

func (r *SettingsRepo) GetForUpdate(ctx context.Context) (*domain.Settings, error) {
	s, err := GetOrNotFound(func() (sqlc.AppSettings, error) { return r.Q(ctx).GetAppSettingsForUpdate(ctx) },
		apperr.ErrAppSettingsNotFound, "SettingsRepo - GetForUpdate")
	if err != nil {
		return nil, err
	}

	return toDomainAppSettings(s), nil
}

// ReconcileStartupDefaults syncs environment-derived startup defaults into the
// singleton app_settings row only while it still contains the migration default
// app name, preserving later admin edits.
func (r *SettingsRepo) ReconcileStartupDefaults(
	ctx context.Context,
	appName string,
	fromName string,
	fromEmail string,
	resendEnabled bool,
	githubEnabled bool,
	googleEnabled bool,
	defaultAppName string,
) error {
	if err := r.Q(ctx).ReconcileAppSettingsStartupDefaults(ctx, sqlc.ReconcileAppSettingsStartupDefaultsParams{
		AppName:            appName,
		ResendFromName:     fromName,
		ResendFromEmail:    fromEmail,
		ResendEnabled:      resendEnabled,
		OAuthGithubEnabled: githubEnabled,
		OAuthGoogleEnabled: googleEnabled,
		AppName_2:          defaultAppName,
	}); err != nil {
		return fmt.Errorf("SettingsRepo - ReconcileStartupDefaults: %w", err)
	}

	return nil
}

// settingsIntFields returns all int-typed Settings fields as an ordered slice for
// batch narrowing via ConvertIntFieldsToInt32. The positional order (vals[0..19])
// must stay in sync with the index-based destructuring in Update.
func settingsIntFields(s *domain.Settings) []IntField {
	return []IntField{
		{"VerifyTTLHours", s.VerifyTTLHours},
		{"ResetTTLHours", s.ResetTTLHours},
		{"SubmitLimitPerUser", s.SubmitLimitPerUser},
		{"SubmitLimitDurationMin", s.SubmitLimitDurationMin},
		{"DefaultPerPage", s.DefaultPerPage},
		{"MaxPerPage", s.MaxPerPage},
		{"CSVExportMaxRows", s.CSVExportMaxRows},
		{"RateLimitLoginPerMinute", s.RateLimitLoginPerMinute},
		{"RateLimitRegisterPerMinute", s.RateLimitRegisterPerMinute},
		{"RateLimitForgotPasswordPerMinute", s.RateLimitForgotPasswordPerMinute},
		{"RateLimitResetPasswordPerMinute", s.RateLimitResetPasswordPerMinute},
		{"RateLimitLogoutPerMinute", s.RateLimitLogoutPerMinute},
		{"RateLimitRefreshPerMinute", s.RateLimitRefreshPerMinute},
		{"RateLimitScoreboardPerMinute", s.RateLimitScoreboardPerMinute},
		{"RateLimitGeneralIPPerMinute", s.RateLimitGeneralIPPerMinute},
		{"RateLimitVerifyEmailPerMinute", s.RateLimitVerifyEmailPerMinute},
		{"RateLimitOAuthCallbackPerMinute", s.RateLimitOAuthCallbackPerMinute},
		{"RateLimitOAuthRedirectPerMinute", s.RateLimitOAuthRedirectPerMinute},
		{"RateLimitCommentPerMinute", s.RateLimitCommentPerMinute},
		{"MaxTeams", s.MaxTeams},
	}
}

// Update writes all Settings fields to app_settings. Integer fields are narrowed from int to
// int32 via settingsIntFields; vals are destructured positionally (vals[0..19] mapping must
// match the settingsIntFields order).
func (r *SettingsRepo) Update(ctx context.Context, s *domain.Settings) error {
	vals, err := ConvertIntFieldsToInt32(settingsIntFields(s))
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update: %w", err)
	}

	verifyTTL, resetTTL, submitLimit, submitDuration := vals[0], vals[1], vals[2], vals[3]
	defaultPerPage, maxPerPage, csvExportMaxRows := vals[4], vals[5], vals[6]
	rateLimitLogin, rateLimitRegister, rateLimitForgotPassword, rateLimitResetPassword := vals[7], vals[8], vals[9], vals[10]
	rateLimitLogout, rateLimitRefresh, rateLimitScoreboard, rateLimitGeneralIP := vals[11], vals[12], vals[13], vals[14]
	rateLimitVerifyEmail, rateLimitOAuthCallback, rateLimitOAuthRedirect, rateLimitComment := vals[15], vals[16], vals[17], vals[18]
	maxTeams := vals[19]
	now := time.Now()

	err = r.Q(ctx).UpdateAppSettings(ctx, sqlc.UpdateAppSettingsParams{
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
		UpdatedAt:                        pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		return fmt.Errorf("SettingsRepo - Update: %w", err)
	}

	return nil
}

// UpdateIfCurrent performs an optimistic-concurrency update: the SQL WHERE
// clause matches on updated_at so the write is rejected (ErrSettingsConflict)
// when another transaction has already committed a newer version.
func (r *SettingsRepo) UpdateIfCurrent(ctx context.Context, s *domain.Settings) error {
	vals, err := ConvertIntFieldsToInt32(settingsIntFields(s))
	if err != nil {
		return fmt.Errorf("SettingsRepo - UpdateIfCurrent: %w", err)
	}

	verifyTTL, resetTTL, submitLimit, submitDuration := vals[0], vals[1], vals[2], vals[3]
	defaultPerPage, maxPerPage, csvExportMaxRows := vals[4], vals[5], vals[6]
	rateLimitLogin, rateLimitRegister, rateLimitForgotPassword, rateLimitResetPassword := vals[7], vals[8], vals[9], vals[10]
	rateLimitLogout, rateLimitRefresh, rateLimitScoreboard, rateLimitGeneralIP := vals[11], vals[12], vals[13], vals[14]
	rateLimitVerifyEmail, rateLimitOAuthCallback, rateLimitOAuthRedirect, rateLimitComment := vals[15], vals[16], vals[17], vals[18]
	maxTeams := vals[19]
	now := time.Now()

	_, err = r.Q(ctx).UpdateAppSettingsIfCurrent(ctx, sqlc.UpdateAppSettingsIfCurrentParams{
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
		UpdatedAt:                        pgutil.TimeToTimestamptz(&now),
		UpdatedAt_2:                      pgutil.TimeToTimestamptz(&s.UpdatedAt),
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return apperr.ErrSettingsConflict
		}

		return fmt.Errorf("SettingsRepo - UpdateIfCurrent: %w", err)
	}

	return nil
}
