package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

const competitionCacheKey = "competition"

type CompetitionUseCase struct {
	competitionRepo repo.CompetitionRepository
	auditLogRepo    repo.AuditLogRepository
	redis           *redis.Client
}

func NewCompetitionUseCase(competitionRepo repo.CompetitionRepository, auditLogRepo repo.AuditLogRepository, redis *redis.Client) *CompetitionUseCase {
	return &CompetitionUseCase{
		competitionRepo: competitionRepo,
		auditLogRepo:    auditLogRepo,
		redis:           redis,
	}
}

func (uc *CompetitionUseCase) Get(ctx context.Context) (*entity.Competition, error) {
	val, err := uc.redis.Get(ctx, competitionCacheKey).Result()
	if err == nil {
		var comp entity.Competition
		if err := json.Unmarshal([]byte(val), &comp); err == nil {
			return &comp, nil
		}
	}

	comp, err := uc.competitionRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionUseCase - Get: %w", err)
	}

	if bytes, err := json.Marshal(comp); err == nil {
		uc.redis.Set(ctx, competitionCacheKey, bytes, 5*time.Second)
	}

	return comp, nil
}

func (uc *CompetitionUseCase) Update(ctx context.Context, comp *entity.Competition, actorId uuid.UUID, clientIP string) error {
	err := uc.competitionRepo.Update(ctx, comp)
	if err != nil {
		return fmt.Errorf("CompetitionUseCase - Update: %w", err)
	}

	uc.redis.Del(ctx, competitionCacheKey)

	auditLog := &entity.AuditLog{
		UserId:     &actorId,
		Action:     entity.AuditActionUpdate,
		EntityType: entity.AuditEntityCompetition,
		EntityId:   "settings",
		IP:         clientIP,
		Details: map[string]any{
			"message": "competition settings updated",
		},
	}
	_ = uc.auditLogRepo.Create(ctx, auditLog)

	return nil
}

func (uc *CompetitionUseCase) GetStatus(ctx context.Context) (entity.CompetitionStatus, error) {
	comp, err := uc.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("CompetitionUseCase - GetStatus: %w", err)
	}

	return comp.GetStatus(), nil
}

func (uc *CompetitionUseCase) IsSubmissionAllowed(ctx context.Context) (bool, error) {
	comp, err := uc.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("CompetitionUseCase - IsSubmissionAllowed: %w", err)
	}

	return comp.IsSubmissionAllowed(), nil
}
