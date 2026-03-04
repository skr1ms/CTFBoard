package competition

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

const (
	competitionParamCacheTTL = 30 * time.Second
	loadAllKey               = "competition_params:loadAll"
)

type CompetitionParamUseCase struct {
	deps     CompetitionParamDeps
	cache    map[string]*entity.CompetitionParam
	mu       sync.RWMutex
	lastLoad time.Time
	sf       singleflight.Group
}

type CompetitionParamDeps struct {
	Repo         repo.CompetitionParamRepository
	AuditLogRepo repo.AuditLogRepository
	TM           repo.TransactionManager
	Logger       logger.Logger
}

var _ usecase.CompetitionParamUseCase = (*CompetitionParamUseCase)(nil)

func NewCompetitionParamUseCase(deps CompetitionParamDeps) *CompetitionParamUseCase {
	if deps.Logger == nil {
		deps.Logger = logger.Noop()
	}
	return &CompetitionParamUseCase{
		deps:  deps,
		cache: make(map[string]*entity.CompetitionParam),
	}
}

func (uc *CompetitionParamUseCase) invalidate() {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.lastLoad = time.Time{}
}

func (uc *CompetitionParamUseCase) loadAll(ctx context.Context) error {
	params, err := uc.deps.Repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("CompetitionParamUseCase - loadAll - CompetitionParamRepo.GetAll: %w", err)
	}
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.cache = make(map[string]*entity.CompetitionParam)
	for _, p := range params {
		uc.cache[p.Key] = p
	}
	uc.lastLoad = time.Now()
	return nil
}

func (uc *CompetitionParamUseCase) Get(ctx context.Context, key string) (*entity.CompetitionParam, error) {
	uc.mu.RLock()
	cacheValid := time.Since(uc.lastLoad) < competitionParamCacheTTL
	if cacheValid {
		if p, ok := uc.cache[key]; ok {
			uc.mu.RUnlock()
			return p, nil
		}
	}
	uc.mu.RUnlock()

	if !cacheValid {
		loadCtx := context.WithoutCancel(ctx)
		if _, err, _ := uc.sf.Do(loadAllKey, func() (any, error) {
			return nil, uc.loadAll(loadCtx)
		}); err != nil {
			return nil, fmt.Errorf("CompetitionParamUseCase - Get - loadAll: %w", err)
		}
	}

	uc.mu.RLock()
	p, ok := uc.cache[key]
	uc.mu.RUnlock()
	if ok {
		return p, nil
	}

	sfKey := "competition_params:key:" + key
	v, err, _ := uc.sf.Do(sfKey, func() (any, error) {
		c, err := uc.deps.Repo.GetByKey(context.WithoutCancel(ctx), key)
		if err != nil {
			return nil, fmt.Errorf("CompetitionParamUseCase - Get - CompetitionParamRepo.GetByKey: %w", err)
		}
		uc.mu.Lock()
		uc.cache[key] = c
		uc.mu.Unlock()
		return c, nil
	})
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamUseCase - Get - singleflight: %w", err)
	}
	c, ok := v.(*entity.CompetitionParam)
	if !ok {
		return nil, fmt.Errorf("CompetitionParamUseCase - Get: unexpected cache type for key %q", key)
	}
	return c, nil
}

func (uc *CompetitionParamUseCase) GetAll(ctx context.Context) ([]*entity.CompetitionParam, error) {
	uc.mu.RLock()
	cacheValid := time.Since(uc.lastLoad) < competitionParamCacheTTL
	if cacheValid {
		list := make([]*entity.CompetitionParam, 0, len(uc.cache))
		for _, p := range uc.cache {
			list = append(list, p)
		}
		uc.mu.RUnlock()
		return list, nil
	}
	uc.mu.RUnlock()

	loadCtx := context.WithoutCancel(ctx)
	if _, err, _ := uc.sf.Do(loadAllKey, func() (any, error) {
		return nil, uc.loadAll(loadCtx)
	}); err != nil {
		return nil, fmt.Errorf("CompetitionParamUseCase - GetAll - loadAll: %w", err)
	}

	uc.mu.RLock()
	list := make([]*entity.CompetitionParam, 0, len(uc.cache))
	for _, p := range uc.cache {
		list = append(list, p)
	}
	uc.mu.RUnlock()
	return list, nil
}

func (uc *CompetitionParamUseCase) Set(ctx context.Context, key, value, description string, valueType entity.CompetitionParamValueType, actorID uuid.UUID, clientIP string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return httperr.ErrCompetitionParamKeyRequired
	}
	if err := uc.validateValueType(valueType, value); err != nil {
		return fmt.Errorf("CompetitionParamUseCase - Set - validateValueType: %w", err)
	}
	p := &entity.CompetitionParam{
		Key:         key,
		Value:       value,
		ValueType:   valueType,
		Description: description,
	}
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.Repo.Upsert(ctx, p); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - Set - CompetitionParamRepo.Upsert: %w", err)
		}
		auditLog := &entity.AuditLog{
			UserID:     &actorID,
			Action:     entity.AuditActionUpdate,
			EntityType: "competition_param",
			EntityID:   key,
			IP:         clientIP,
			Details: map[string]any{
				"message": "competition param updated",
				"key":     key,
			},
		}
		if err := uc.deps.AuditLogRepo.Create(ctx, auditLog); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - Set - AuditLogRepo.Create: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("CompetitionParamUseCase - Set - TM.Run: %w", err)
	}
	uc.invalidate()
	return nil
}

func (uc *CompetitionParamUseCase) Delete(ctx context.Context, key string, actorID uuid.UUID, clientIP string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return httperr.ErrCompetitionParamKeyRequired
	}
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.Repo.GetByKey(ctx, key); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - Delete - CompetitionParamRepo.GetByKey: %w", err)
		}
		if err := uc.deps.Repo.Delete(ctx, key); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - Delete - CompetitionParamRepo.Delete: %w", err)
		}
		auditLog := &entity.AuditLog{
			UserID:     &actorID,
			Action:     entity.AuditActionDelete,
			EntityType: "competition_param",
			EntityID:   key,
			IP:         clientIP,
			Details: map[string]any{
				"message": "competition param deleted",
				"key":     key,
			},
		}
		if err := uc.deps.AuditLogRepo.Create(ctx, auditLog); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - Delete - AuditLogRepo.Create: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("CompetitionParamUseCase - Delete - TM.Run: %w", err)
	}
	uc.invalidate()
	return nil
}

func (uc *CompetitionParamUseCase) validateValueType(valueType entity.CompetitionParamValueType, value string) error {
	switch valueType {
	case entity.CompetitionParamTypeInt:
		if _, err := strconv.Atoi(value); err != nil {
			return httperr.ErrCompetitionParamInvalidValueType
		}
	case entity.CompetitionParamTypeBool:
		if value != "true" && value != "false" {
			return httperr.ErrCompetitionParamInvalidValueType
		}
	case entity.CompetitionParamTypeString, entity.CompetitionParamTypeJSON:
	default:
		return httperr.ErrCompetitionParamInvalidValueType
	}
	return nil
}

func (uc *CompetitionParamUseCase) GetString(ctx context.Context, key, defaultVal string) string {
	p, err := uc.Get(ctx, key)
	if err != nil {
		if errors.Is(err, httperr.ErrCompetitionParamNotFound) {
			return defaultVal
		}
		uc.deps.Logger.WithError(err).Warn("competition_params: GetString failed, returning default", logger.Fields{"key": key})
		return defaultVal
	}
	if p == nil {
		return defaultVal
	}
	return p.Value
}

func (uc *CompetitionParamUseCase) GetInt(ctx context.Context, key string, defaultVal int) int {
	p, err := uc.Get(ctx, key)
	if err != nil {
		if errors.Is(err, httperr.ErrCompetitionParamNotFound) {
			return defaultVal
		}
		uc.deps.Logger.WithError(err).Warn("competition_params: GetInt failed, returning default", logger.Fields{"key": key})
		return defaultVal
	}
	if p == nil {
		return defaultVal
	}
	val, err := strconv.Atoi(p.Value)
	if err != nil {
		return defaultVal
	}
	return val
}

func (uc *CompetitionParamUseCase) GetBool(ctx context.Context, key string, defaultVal bool) bool {
	p, err := uc.Get(ctx, key)
	if err != nil {
		if errors.Is(err, httperr.ErrCompetitionParamNotFound) {
			return defaultVal
		}
		uc.deps.Logger.WithError(err).Warn("competition_params: GetBool failed, returning default", logger.Fields{"key": key})
		return defaultVal
	}
	if p == nil {
		return defaultVal
	}
	return p.Value == "true"
}
