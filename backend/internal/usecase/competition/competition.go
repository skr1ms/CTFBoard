package competition

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
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

const (
	localCompTTL  = 5 * time.Second
	redisCacheTTL = 30 * time.Second
)

type CompetitionUseCase struct {
	deps        CompetitionDeps
	sf          singleflight.Group // zero value is valid, no explicit init required
	localComp   atomic.Pointer[entity.Competition]
	localCompAt atomic.Int64 // unix nano of last store
}

type CompetitionDeps struct {
	CompetitionRepo repo.CompetitionRepository
	AuditLogRepo    repo.AuditLogRepository
	TM              repo.TransactionManager
	Redis           cache.KeyValueStore
	Logger          logger.Logger
}

var _ usecase.CompetitionUseCase = (*CompetitionUseCase)(nil)

func NewCompetitionUseCase(deps CompetitionDeps) *CompetitionUseCase {
	if deps.Logger == nil {
		deps.Logger = logger.Noop()
	}
	return &CompetitionUseCase{deps: deps}
}

//nolint:gocognit // multi-layer cache (local → Redis → DB) with frozen-scoreboard branching
func (uc *CompetitionUseCase) Get(ctx context.Context) (*entity.Competition, error) {
	// Fast path: in-process cache avoids Redis round-trip on every middleware call.
	if cached := uc.localComp.Load(); cached != nil {
		age := time.Duration(time.Now().UnixNano() - uc.localCompAt.Load())
		if age < localCompTTL {
			return cached, nil
		}
	}

	if uc.deps.Redis != nil {
		val, err := uc.deps.Redis.Get(ctx, cache.KeyCompetition)
		if err == nil {
			var comp entity.Competition
			if err := json.Unmarshal([]byte(val), &comp); err == nil {
				uc.storeLocal(&comp)
				return &comp, nil
			}
		}
	}

	v, err, _ := uc.sf.Do(cache.KeyCompetition, func() (any, error) {
		comp, err := uc.deps.CompetitionRepo.Get(context.WithoutCancel(ctx))
		if err != nil {
			return nil, fmt.Errorf("CompetitionUseCase - Get - CompetitionRepo.Get: %w", err)
		}

		if uc.deps.Redis != nil {
			if bytes, err := json.Marshal(comp); err == nil {
				_ = uc.deps.Redis.Set(context.WithoutCancel(ctx), cache.KeyCompetition, bytes, redisCacheTTL) //nolint:errcheck
			}
		}
		return comp, nil
	})
	if err != nil {
		return nil, fmt.Errorf("CompetitionUseCase - Get - CompetitionRepo.Get: %w", err)
	}
	comp, ok := v.(*entity.Competition)
	if !ok {
		return nil, fmt.Errorf("CompetitionUseCase - Get: unexpected type")
	}
	uc.storeLocal(comp)
	return comp, nil
}

func (uc *CompetitionUseCase) storeLocal(comp *entity.Competition) {
	c := *comp
	uc.localComp.Store(&c)
	uc.localCompAt.Store(time.Now().UnixNano())
}

func timePtrEqual(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	ta, tb := a.Truncate(time.Second), b.Truncate(time.Second)
	return ta.Equal(tb)
}

//nolint:gocyclo,gocognit // status and field checks
func (uc *CompetitionUseCase) Update(ctx context.Context, comp *entity.Competition, actorID uuid.UUID, clientIP string) error {
	current, err := uc.deps.CompetitionRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("CompetitionUseCase - Update - CompetitionRepo.Get: %w", err)
	}

	// MinTeamSize and MaxTeamSize are not part of UpdateCompetitionRequest (they are
	// configured at startup via env). Preserve the current DB values to avoid zeroing
	// them out on every PUT /admin/competition.
	if comp.MinTeamSize == 0 {
		comp.MinTeamSize = current.MinTeamSize
	}
	if comp.MaxTeamSize == 0 {
		comp.MaxTeamSize = current.MaxTeamSize
	}

	if comp.Mode != "" && !comp.Mode.IsValid() {
		return httperr.NewValidationErrorf("invalid competition mode %q: must be solo_only, teams_only, or flexible", comp.Mode)
	}
	if comp.Mode == "" {
		comp.Mode = current.Mode
	}

	status := current.GetStatus()
	if status == entity.CompetitionStatusActive || status == entity.CompetitionStatusFrozen || status == entity.CompetitionStatusPaused {
		if comp.StartTime == nil {
			comp.StartTime = current.StartTime
		}
		if comp.EndTime == nil {
			comp.EndTime = current.EndTime
		}
		if comp.FreezeTime == nil {
			comp.FreezeTime = current.FreezeTime
		}
		if comp.Mode != current.Mode ||
			comp.AllowTeamSwitch != current.AllowTeamSwitch ||
			!timePtrEqual(comp.StartTime, current.StartTime) ||
			!timePtrEqual(comp.EndTime, current.EndTime) ||
			!timePtrEqual(comp.FreezeTime, current.FreezeTime) {
			return httperr.ErrCompetitionActiveCannotUpdate
		}
	}
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
	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.CompetitionRepo.Update(ctx, comp); err != nil {
			return fmt.Errorf("CompetitionUseCase - Update - CompetitionRepo.Update: %w", err)
		}
		if err := uc.deps.AuditLogRepo.Create(ctx, auditLog); err != nil {
			return fmt.Errorf("CompetitionUseCase - Update - AuditLogRepo.Create: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("CompetitionUseCase - Update - TM.Run: %w", err)
	}
	// Invalidate in-process cache so the next Get() re-fetches fresh data.
	uc.localComp.Store(nil)
	uc.localCompAt.Store(0)
	if uc.deps.Redis != nil {
		if err := uc.deps.Redis.Del(ctx, cache.KeyCompetition); err != nil {
			uc.deps.Logger.WithError(err).Warn("CompetitionUseCase - Update: failed to invalidate cache; stale data for up to 5s")
		}
	}
	return nil
}

func (uc *CompetitionUseCase) GetStatus(ctx context.Context) (entity.CompetitionStatus, error) {
	comp, err := uc.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("CompetitionUseCase - GetStatus - Get: %w", err)
	}

	return comp.GetStatus(), nil
}

func (uc *CompetitionUseCase) IsSubmissionAllowed(ctx context.Context) (bool, error) {
	comp, err := uc.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("CompetitionUseCase - IsSubmissionAllowed - Get: %w", err)
	}

	return comp.IsSubmissionAllowed(), nil
}
