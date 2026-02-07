package competition

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/cache"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
	"golang.org/x/sync/singleflight"
)

type CompetitionUseCase struct {
	competitionRepo repo.CompetitionRepository
	auditLogRepo    repo.AuditLogRepository
	redis           *redis.Client
	sf              singleflight.Group
}

func NewCompetitionUseCase(
	competitionRepo repo.CompetitionRepository,
	auditLogRepo repo.AuditLogRepository,
	redis *redis.Client,
) *CompetitionUseCase {
	return &CompetitionUseCase{
		competitionRepo: competitionRepo,
		auditLogRepo:    auditLogRepo,
		redis:           redis,
	}
}

func (uc *CompetitionUseCase) Get(ctx context.Context) (*entity.Competition, error) {
	val, err := uc.redis.Get(ctx, cache.KeyCompetition).Result()
	if err == nil {
		var comp entity.Competition
		if err := json.Unmarshal([]byte(val), &comp); err == nil {
			return &comp, nil
		}
	}

	v, err, _ := uc.sf.Do(cache.KeyCompetition, func() (any, error) {
		comp, err := uc.competitionRepo.Get(ctx)
		if err != nil {
			return nil, err
		}
		if bytes, err := json.Marshal(comp); err == nil {
			uc.redis.Set(context.Background(), cache.KeyCompetition, bytes, 5*time.Second)
		}
		return comp, nil
	})
	if err != nil {
		return nil, usecaseutil.Wrap(err, "CompetitionUseCase - Get")
	}
	comp, ok := v.(*entity.Competition)
	if !ok {
		return nil, usecaseutil.Wrap(fmt.Errorf("unexpected type"), "CompetitionUseCase - Get")
	}
	return comp, nil
}

func (uc *CompetitionUseCase) Update(ctx context.Context, comp *entity.Competition, actorID uuid.UUID, clientIP string) error {
	err := uc.competitionRepo.Update(ctx, comp)
	if err != nil {
		return usecaseutil.Wrap(err, "CompetitionUseCase - Update")
	}

	uc.redis.Del(ctx, cache.KeyCompetition)

	auditLog := &entity.AuditLog{
		UserID:     &actorID,
		Action:     entity.AuditActionUpdate,
		EntityType: entity.AuditEntityCompetition,
		EntityID:   "settings",
		IP:         clientIP,
		Details: map[string]any{
			"message": "competition settings updated",
		},
	}
	if err := uc.auditLogRepo.Create(ctx, auditLog); err != nil {
		return usecaseutil.Wrap(err, "CompetitionUseCase - Update - Create")
	}
	return nil
}

func (uc *CompetitionUseCase) GetStatus(ctx context.Context) (entity.CompetitionStatus, error) {
	comp, err := uc.Get(ctx)
	if err != nil {
		return "", usecaseutil.Wrap(err, "CompetitionUseCase - GetStatus")
	}

	return comp.GetStatus(), nil
}

func (uc *CompetitionUseCase) IsSubmissionAllowed(ctx context.Context) (bool, error) {
	comp, err := uc.Get(ctx)
	if err != nil {
		return false, usecaseutil.Wrap(err, "CompetitionUseCase - IsSubmissionAllowed")
	}

	return comp.IsSubmissionAllowed(), nil
}
