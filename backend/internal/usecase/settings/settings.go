package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	redisKeys "github.com/skr1ms/CTFBoard/pkg/redis"
)

const cacheTTL = 5 * time.Minute

type SettingsUseCase struct {
	repo         repo.AppSettingsRepository
	auditLogRepo repo.AuditLogRepository
	redis        *redis.Client
}

func NewSettingsUseCase(
	repo repo.AppSettingsRepository,
	auditLogRepo repo.AuditLogRepository,
	redis *redis.Client,
) *SettingsUseCase {
	return &SettingsUseCase{
		repo:         repo,
		auditLogRepo: auditLogRepo,
		redis:        redis,
	}
}

func (uc *SettingsUseCase) Get(ctx context.Context) (*entity.AppSettings, error) {
	val, err := uc.redis.Get(ctx, redisKeys.KeyAppSettings).Result()
	if err == nil {
		var s entity.AppSettings
		if err := json.Unmarshal([]byte(val), &s); err == nil {
			return &s, nil
		}
	}

	s, err := uc.repo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("SettingsUseCase - Get: %w", err)
	}

	if bytes, err := json.Marshal(s); err == nil {
		uc.redis.Set(ctx, redisKeys.KeyAppSettings, bytes, cacheTTL)
	}

	return s, nil
}

func (uc *SettingsUseCase) Update(ctx context.Context, s *entity.AppSettings, actorID uuid.UUID, clientIP string) error {
	if err := uc.validate(s); err != nil {
		return err
	}

	if err := uc.repo.Update(ctx, s); err != nil {
		return fmt.Errorf("SettingsUseCase - Update: %w", err)
	}

	uc.redis.Del(ctx, redisKeys.KeyAppSettings)

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
	if err := uc.auditLogRepo.Create(ctx, auditLog); err != nil {
		return fmt.Errorf("SettingsUseCase - Update - Create audit: %w", err)
	}

	return nil
}

func (uc *SettingsUseCase) validate(s *entity.AppSettings) error {
	if s.SubmitLimitPerUser < 1 {
		return fmt.Errorf("SettingsUseCase - Update: submit_limit_per_user must be >= 1")
	}
	if s.SubmitLimitDurationMin < 1 {
		return fmt.Errorf("SettingsUseCase - Update: submit_limit_duration_min must be >= 1")
	}
	if s.VerifyTTLHours < 1 || s.VerifyTTLHours > 168 {
		return fmt.Errorf("SettingsUseCase - Update: verify_ttl_hours must be between 1 and 168")
	}
	if s.ResetTTLHours < 1 || s.ResetTTLHours > 168 {
		return fmt.Errorf("SettingsUseCase - Update: reset_ttl_hours must be between 1 and 168")
	}
	switch s.ScoreboardVisible {
	case entity.ScoreboardVisiblePublic, entity.ScoreboardVisibleHidden, entity.ScoreboardVisibleAdminsOnly:
	default:
		return fmt.Errorf("SettingsUseCase - Update: scoreboard_visible must be public, hidden, or admins_only")
	}
	return nil
}
