package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const (
	cacheTTL           = 5 * time.Minute
	settingsInvChannel = "settings:inv"
	invTimeout         = 2 * time.Second
	subBackoffInitial  = 1 * time.Second
	subBackoffMax      = 30 * time.Second
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

		go uc.subscribeInvalidation(stopCtx)
	}

	return uc
}

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
						return httperr.ErrSettingsCannotChangeDuringCompetition
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

func (uc *SettingsUseCase) subscribeInvalidation(stopCtx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = subBackoffInitial
	bo.MaxInterval = subBackoffMax
	bo.MaxElapsedTime = 0
	notify := func(err error, d time.Duration) {
		uc.deps.Logger.WithError(err).Warn("settings: subscribe to invalidation channel failed, retrying")
	}

	for {
		select {
		case <-stopCtx.Done():
			return
		default:
		}

		var ch <-chan string

		op := func() error {
			var err error

			ch, err = uc.deps.PubSub.Subscribe(stopCtx, settingsInvChannel)

			return err
		}
		if err := backoff.RetryNotify(op, backoff.WithContext(bo, stopCtx), notify); err != nil {
			return
		}

		bo.Reset()

	readLoop:
		for {
			select {
			case <-stopCtx.Done():
				return
			case _, ok := <-ch:
				if !ok {
					break readLoop
				}

				uc.sf.Forget(cache.KeyAppSettings)
			}
		}

		next := bo.NextBackOff()
		if next == backoff.Stop {
			return
		}

		t := time.NewTimer(next)
		select {
		case <-stopCtx.Done():
			t.Stop()

			return
		case <-t.C:
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
		return httperr.NewValidationErrorf("scoreboard_visible must be public, hidden, or admins_only")
	}

	return nil
}

func validateTimings(s *domain.Settings) error {
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

func validatePagination(s *domain.Settings) error {
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

func validateRateLimits(s *domain.Settings) error {
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

	if s.RateLimitOAuthRedirectPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_oauth_redirect_per_minute must be >= 1")
	}

	if s.RateLimitCommentPerMinute < 1 {
		return httperr.NewValidationErrorf("rate_limit_comment_per_minute must be >= 1")
	}

	return nil
}
