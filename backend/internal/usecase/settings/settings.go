package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const (
	cacheTTL           = 5 * time.Minute
	settingsInvChannel = "settings:inv"
	invTimeout         = 2 * time.Second

	maxTTLHours     = 168
	maxPerPageLimit = 1000
)

type SettingsUseCase struct {
	deps SettingsDeps
	sf   singleflight.Group
}

type SettingsDeps struct {
	Repo         repo.SettingsRepository
	AuditLogRepo repo.AuditLogRepository
	TM           repo.TransactionManager
	Redis        cachekit.KeyValueStore
	CompRepo     repo.CompetitionRepository
	ConfigUC     usecase.CompetitionParamUseCase
	PubSub       cachekit.PubSubStore
	StopContext  context.Context
	Logger       logkit.Logger
}

var _ usecase.SettingsUseCase = (*SettingsUseCase)(nil)

func NewSettingsUseCase(deps SettingsDeps) *SettingsUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	uc := &SettingsUseCase{deps: deps}

	if deps.PubSub != nil {
		stopCtx := deps.StopContext
		if stopCtx == nil {
			stopCtx = context.Background()
		}

		go cache.SubscribeInvalidation(stopCtx, deps.PubSub, settingsInvChannel, func() {
			uc.sf.Forget(cache.KeyAppSettings)
		}, deps.Logger, "settings")
	}

	return uc
}

// Get returns the current application settings using a two-layer cache.
// A Redis entry is checked first; on miss, concurrent lookups are deduplicated
// via singleflight and the result is fetched from the database and written
// back to Redis for cacheTTL. The singleflight key is the same as the Redis
// key so a PubSub-driven invalidation (sf.Forget) also prevents stale reads.
func (uc *SettingsUseCase) Get(ctx context.Context) (*domain.Settings, error) {
	if uc.deps.Redis != nil {
		val, err := uc.deps.Redis.Get(ctx, cache.KeyAppSettings)
		if err == nil {
			var s domain.Settings

			err := json.Unmarshal([]byte(val), &s)
			if err == nil {
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

	s, ok := v.(*domain.Settings)
	if !ok {
		return nil, fmt.Errorf("SettingsUseCase - Get: unexpected type")
	}

	return s, nil
}

// Update validates and persists new application settings inside a transaction.
// GetForUpdate acquires a pessimistic lock on the settings row. When a
// competition is active/frozen/paused, changes to ScoreboardVisible or
// RegistrationOpen are rejected to preserve competition integrity. UpdateIfCurrent
// provides optimistic concurrency: it errors with ErrSettingsConflict if the row
// was modified between the lock and the update. After commit, invalidateCache
// evicts the singleflight entry, deletes the Redis key, and broadcasts a PubSub
// invalidation to other instances.
func (uc *SettingsUseCase) Update(ctx context.Context, s *domain.Settings, actorID uuid.UUID, clientIP string) error {
	err := uc.validate(s)
	if err != nil {
		return fmt.Errorf("SettingsUseCase - Update - validate: %w", err)
	}

	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		current, err := uc.deps.Repo.GetForUpdate(ctx)
		if err != nil {
			return fmt.Errorf("SettingsUseCase - Update - SettingsRepo.GetForUpdate: %w", err)
		}

		if uc.deps.CompRepo != nil {
			comp, err := uc.deps.CompRepo.Get(ctx)
			if err == nil {
				status := comp.GetStatus()
				if status == domain.CompetitionStatusActive || status == domain.CompetitionStatusFrozen || status == domain.CompetitionStatusPaused {
					if s.ScoreboardVisible != current.ScoreboardVisible || s.RegistrationOpen != current.RegistrationOpen {
						return apperr.ErrSettingsCannotChangeDuringCompetition
					}
				}
			}
		}

		if err := uc.deps.Repo.UpdateIfCurrent(ctx, s); err != nil {
			return fmt.Errorf("SettingsUseCase - Update - SettingsRepo.UpdateIfCurrent: %w", err)
		}

		auditLog := &domain.AuditLog{
			UserID:     &actorID,
			Action:     domain.AuditActionUpdate,
			EntityType: domain.AuditEntityAppSettings,
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
	})
	if err != nil {
		return fmt.Errorf("SettingsUseCase - Update - TM.Run: %w", err)
	}

	uc.invalidateCache()

	return nil
}

func (uc *SettingsUseCase) invalidateCache() {
	uc.sf.Forget(cache.KeyAppSettings)

	ctx, cancel := context.WithTimeout(context.Background(), invTimeout)
	defer cancel()

	if uc.deps.Redis != nil {
		_ = uc.deps.Redis.Del(ctx, cache.KeyAppSettings) //nolint:errcheck // best-effort cache invalidation
	}

	if uc.deps.PubSub != nil {
		if err := uc.deps.PubSub.Publish(ctx, settingsInvChannel, "1"); err != nil {
			uc.deps.Logger.WithError(err).Warn("settings: pubsub invalidation broadcast failed")
		}
	}
}

func (uc *SettingsUseCase) validate(s *domain.Settings) error {
	err := validateTimings(s)
	if err != nil {
		return fmt.Errorf("SettingsUseCase - validate - validateTimings: %w", err)
	}

	err = validatePagination(s)
	if err != nil {
		return fmt.Errorf("SettingsUseCase - validate - validatePagination: %w", err)
	}

	err = validateRateLimits(s)
	if err != nil {
		return fmt.Errorf("SettingsUseCase - validate - validateRateLimits: %w", err)
	}

	switch s.ScoreboardVisible {
	case domain.ScoreboardVisiblePublic, domain.ScoreboardVisibleHidden, domain.ScoreboardVisibleAdminsOnly:
	default:
		return apperr.NewValidationErrorf("scoreboard_visible must be public, hidden, or admins_only")
	}

	return nil
}

func validateTimings(s *domain.Settings) error {
	if s.SubmitLimitPerUser < 1 {
		return apperr.NewValidationErrorf("submit_limit_per_user must be >= 1")
	}

	if s.SubmitLimitDurationMin < 1 {
		return apperr.NewValidationErrorf("submit_limit_duration_min must be >= 1")
	}

	if s.VerifyTTLHours < 1 || s.VerifyTTLHours > maxTTLHours {
		return apperr.NewValidationErrorf("verify_ttl_hours must be between 1 and %d", maxTTLHours)
	}

	if s.ResetTTLHours < 1 || s.ResetTTLHours > maxTTLHours {
		return apperr.NewValidationErrorf("reset_ttl_hours must be between 1 and %d", maxTTLHours)
	}

	return nil
}

func validatePagination(s *domain.Settings) error {
	if s.DefaultPerPage < 1 || s.DefaultPerPage > maxPerPageLimit {
		return apperr.NewValidationErrorf("default_per_page must be between 1 and %d", maxPerPageLimit)
	}

	if s.MaxPerPage < 1 || s.MaxPerPage > maxPerPageLimit {
		return apperr.NewValidationErrorf("max_per_page must be between 1 and %d", maxPerPageLimit)
	}

	if s.DefaultPerPage > s.MaxPerPage {
		return apperr.NewValidationErrorf("default_per_page must be <= max_per_page")
	}

	if s.CSVExportMaxRows < 1 {
		return apperr.NewValidationErrorf("csv_export_max_rows must be >= 1")
	}

	return nil
}

func validateRateLimits(s *domain.Settings) error {
	if s.RateLimitLoginPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_login_per_minute must be >= 1")
	}

	if s.RateLimitRegisterPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_register_per_minute must be >= 1")
	}

	if s.RateLimitForgotPasswordPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_forgot_password_per_minute must be >= 1")
	}

	if s.RateLimitResetPasswordPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_reset_password_per_minute must be >= 1")
	}

	if s.RateLimitLogoutPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_logout_per_minute must be >= 1")
	}

	if s.RateLimitRefreshPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_refresh_per_minute must be >= 1")
	}

	if s.RateLimitScoreboardPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_scoreboard_per_minute must be >= 1")
	}

	if s.RateLimitGeneralIPPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_general_ip_per_minute must be >= 1")
	}

	if s.RateLimitVerifyEmailPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_verify_email_per_minute must be >= 1")
	}

	if s.RateLimitOAuthCallbackPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_oauth_callback_per_minute must be >= 1")
	}

	if s.RateLimitOAuthRedirectPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_oauth_redirect_per_minute must be >= 1")
	}

	if s.RateLimitCommentPerMinute < 1 {
		return apperr.NewValidationErrorf("rate_limit_comment_per_minute must be >= 1")
	}

	return nil
}
