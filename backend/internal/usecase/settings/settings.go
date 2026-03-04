package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

const cacheTTL = 5 * time.Minute

type SettingsUseCase struct {
	deps SettingsDeps
	sf   singleflight.Group
}

type SettingsDeps struct {
	Repo         repo.SettingsRepository
	AuditLogRepo repo.AuditLogRepository
	TM           repo.TransactionManager
	Redis        cache.KeyValueStore
	CompRepo     repo.CompetitionRepository
	Logger       logger.Logger
}

var _ usecase.SettingsUseCase = (*SettingsUseCase)(nil)

func NewSettingsUseCase(deps SettingsDeps) *SettingsUseCase {
	if deps.Logger == nil {
		deps.Logger = logger.Noop()
	}
	return &SettingsUseCase{deps: deps}
}

func (uc *SettingsUseCase) Get(ctx context.Context) (*entity.Settings, error) {
	if uc.deps.Redis != nil {
		val, err := uc.deps.Redis.Get(ctx, cache.KeyAppSettings)
		if err == nil {
			var s entity.Settings
			if err := json.Unmarshal([]byte(val), &s); err == nil {
				return &s, nil
			}
		}
	}

	v, err, _ := uc.sf.Do(cache.KeyAppSettings, func() (any, error) {
		s, err := uc.deps.Repo.Get(context.WithoutCancel(ctx))
		if err != nil {
			return nil, fmt.Errorf("SettingsUseCase - Get - SettingsRepo.Get: %w", err)
		}
		if uc.deps.Redis != nil {
			if bytes, err := json.Marshal(s); err == nil {
				_ = uc.deps.Redis.Set(context.WithoutCancel(ctx), cache.KeyAppSettings, bytes, cacheTTL) //nolint:errcheck // best-effort cache write
			}
		}
		return s, nil
	})
	if err != nil {
		return nil, fmt.Errorf("SettingsUseCase - Get - SettingsRepo.Get: %w", err)
	}
	s, ok := v.(*entity.Settings)
	if !ok {
		return nil, fmt.Errorf("SettingsUseCase - Get: unexpected type")
	}
	return s, nil
}

//nolint:gocognit,gocyclo // validation and competition checks
func (uc *SettingsUseCase) Update(ctx context.Context, s *entity.Settings, actorID uuid.UUID, clientIP string) error {
	current, err := uc.deps.Repo.Get(ctx)
	if err != nil {
		return fmt.Errorf("SettingsUseCase - Update - SettingsRepo.Get: %w", err)
	}

	// Preserve existing rate limit values when not explicitly provided (value == 0).
	mergeRateLimits(s, current)

	if uc.deps.CompRepo != nil {
		comp, err := uc.deps.CompRepo.Get(ctx)
		if err == nil {
			status := comp.GetStatus()
			if status == entity.CompetitionStatusActive || status == entity.CompetitionStatusFrozen || status == entity.CompetitionStatusPaused {
				if s.ScoreboardVisible != current.ScoreboardVisible || s.RegistrationOpen != current.RegistrationOpen {
					return httperr.ErrSettingsCannotChangeDuringCompetition
				}
			}
		}
	}
	if err := uc.validate(s); err != nil {
		return fmt.Errorf("SettingsUseCase - Update - validate: %w", err)
	}
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.Repo.Update(ctx, s); err != nil {
			return fmt.Errorf("SettingsUseCase - Update - SettingsRepo.Update: %w", err)
		}
		auditLog := &entity.AuditLog{
			UserID:     &actorID,
			Action:     entity.AuditActionUpdate,
			EntityType: entity.AuditEntityAppSettings,
			EntityID:   "settings",
			IP:         clientIP,
			Details: map[string]any{
				"message": "app settings updated",
			},
		}
		if err := uc.deps.AuditLogRepo.Create(ctx, auditLog); err != nil {
			return fmt.Errorf("SettingsUseCase - Update - AuditLogRepo.Create: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("SettingsUseCase - Update - TM.Run: %w", err)
	}
	if uc.deps.Redis != nil {
		_ = uc.deps.Redis.Del(ctx, cache.KeyAppSettings) //nolint:errcheck // best-effort cache invalidation
	}
	return nil
}

func (uc *SettingsUseCase) validate(s *entity.Settings) error {
	if err := validateTimings(s); err != nil {
		return fmt.Errorf("SettingsUseCase - validate - validateTimings: %w", err)
	}
	if err := validatePagination(s); err != nil {
		return fmt.Errorf("SettingsUseCase - validate - validatePagination: %w", err)
	}
	if err := validateRateLimits(s); err != nil {
		return fmt.Errorf("SettingsUseCase - validate - validateRateLimits: %w", err)
	}
	switch s.ScoreboardVisible {
	case entity.ScoreboardVisiblePublic, entity.ScoreboardVisibleHidden, entity.ScoreboardVisibleAdminsOnly:
	default:
		return httperr.NewValidationErrorf("scoreboard_visible must be public, hidden, or admins_only")
	}
	return nil
}

func validateTimings(s *entity.Settings) error {
	if s.SubmitLimitPerUser < 1 {
		return httperr.NewValidationErrorf("submit_limit_per_user must be >= 1")
	}
	if s.SubmitLimitDurationMin < 1 {
		return httperr.NewValidationErrorf("submit_limit_duration_min must be >= 1")
	}
	if s.VerifyTTLHours < 1 || s.VerifyTTLHours > 168 {
		return httperr.NewValidationErrorf("verify_ttl_hours must be between 1 and 168")
	}
	if s.ResetTTLHours < 1 || s.ResetTTLHours > 168 {
		return httperr.NewValidationErrorf("reset_ttl_hours must be between 1 and 168")
	}
	return nil
}

func validatePagination(s *entity.Settings) error {
	if s.DefaultPerPage < 1 || s.DefaultPerPage > 1000 {
		return httperr.NewValidationErrorf("default_per_page must be between 1 and 1000")
	}
	if s.MaxPerPage < 1 || s.MaxPerPage > 1000 {
		return httperr.NewValidationErrorf("max_per_page must be between 1 and 1000")
	}
	if s.DefaultPerPage > s.MaxPerPage {
		return httperr.NewValidationErrorf("default_per_page must be <= max_per_page")
	}
	if s.CSVExportMaxRows < 1 {
		return httperr.NewValidationErrorf("csv_export_max_rows must be >= 1")
	}
	return nil
}

func validateRateLimits(s *entity.Settings) error {
	if s.RateLimitLoginPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_login_per_minute must be >= 1")
	}
	if s.RateLimitRegisterPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_register_per_minute must be >= 1")
	}
	if s.RateLimitForgotPasswordPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_forgot_password_per_minute must be >= 1")
	}
	if s.RateLimitResetPasswordPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_reset_password_per_minute must be >= 1")
	}
	if s.RateLimitLogoutPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_logout_per_minute must be >= 1")
	}
	if s.RateLimitRefreshPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_refresh_per_minute must be >= 1")
	}
	if s.RateLimitScoreboardPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_scoreboard_per_minute must be >= 1")
	}
	if s.RateLimitGeneralIPPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_general_ip_per_minute must be >= 1")
	}
	if s.RateLimitVerifyEmailPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_verify_email_per_minute must be >= 1")
	}
	if s.RateLimitOAuthCallbackPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_oauth_callback_per_minute must be >= 1")
	}
	return nil
}

// mergeRateLimits copies rate limit fields from src into dst wherever dst has 0
// (meaning the caller did not explicitly provide that field).
func mergeRateLimits(dst, src *entity.Settings) {
	if dst.RateLimitLoginPerMinute == 0 {
		dst.RateLimitLoginPerMinute = src.RateLimitLoginPerMinute
	}
	if dst.RateLimitRegisterPerMinute == 0 {
		dst.RateLimitRegisterPerMinute = src.RateLimitRegisterPerMinute
	}
	if dst.RateLimitForgotPasswordPerMinute == 0 {
		dst.RateLimitForgotPasswordPerMinute = src.RateLimitForgotPasswordPerMinute
	}
	if dst.RateLimitResetPasswordPerMinute == 0 {
		dst.RateLimitResetPasswordPerMinute = src.RateLimitResetPasswordPerMinute
	}
	if dst.RateLimitLogoutPerMinute == 0 {
		dst.RateLimitLogoutPerMinute = src.RateLimitLogoutPerMinute
	}
	if dst.RateLimitRefreshPerMinute == 0 {
		dst.RateLimitRefreshPerMinute = src.RateLimitRefreshPerMinute
	}
	if dst.RateLimitScoreboardPerMinute == 0 {
		dst.RateLimitScoreboardPerMinute = src.RateLimitScoreboardPerMinute
	}
	if dst.RateLimitGeneralIPPerMinute == 0 {
		dst.RateLimitGeneralIPPerMinute = src.RateLimitGeneralIPPerMinute
	}
	if dst.RateLimitVerifyEmailPerMinute == 0 {
		dst.RateLimitVerifyEmailPerMinute = src.RateLimitVerifyEmailPerMinute
	}
	if dst.RateLimitOAuthCallbackPerMinute == 0 {
		dst.RateLimitOAuthCallbackPerMinute = src.RateLimitOAuthCallbackPerMinute
	}
}
