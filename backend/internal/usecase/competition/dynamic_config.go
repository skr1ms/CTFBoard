package competition

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

const configCacheTTL = 30 * time.Second

type DynamicConfigUseCase struct {
	repo         repo.ConfigRepository
	cache        map[string]*entity.Config
	mu           sync.RWMutex
	lastLoad     time.Time
	auditLogRepo repo.AuditLogRepository
}

func NewDynamicConfigUseCase(
	repo repo.ConfigRepository,
	auditLogRepo repo.AuditLogRepository,
) *DynamicConfigUseCase {
	return &DynamicConfigUseCase{
		repo:         repo,
		cache:        make(map[string]*entity.Config),
		auditLogRepo: auditLogRepo,
	}
}

func (uc *DynamicConfigUseCase) invalidate() {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.lastLoad = time.Time{}
}

func (uc *DynamicConfigUseCase) loadAll(ctx context.Context) error {
	configs, err := uc.repo.GetAll(ctx)
	if err != nil {
		return err
	}
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.cache = make(map[string]*entity.Config)
	for _, cfg := range configs {
		uc.cache[cfg.Key] = cfg
	}
	uc.lastLoad = time.Now()
	return nil
}

func (uc *DynamicConfigUseCase) Get(ctx context.Context, key string) (*entity.Config, error) {
	uc.mu.RLock()
	if time.Since(uc.lastLoad) < configCacheTTL {
		if cfg, ok := uc.cache[key]; ok {
			uc.mu.RUnlock()
			return cfg, nil
		}
		uc.mu.RUnlock()
		cfg, err := uc.repo.GetByKey(ctx, key)
		if err != nil {
			return nil, err
		}
		uc.mu.Lock()
		uc.cache[key] = cfg
		uc.mu.Unlock()
		return cfg, nil
	}
	uc.mu.RUnlock()

	uc.mu.Lock()
	if time.Since(uc.lastLoad) < configCacheTTL {
		if cfg, ok := uc.cache[key]; ok {
			uc.mu.Unlock()
			return cfg, nil
		}
		uc.mu.Unlock()
		cfg, err := uc.repo.GetByKey(ctx, key)
		if err != nil {
			return nil, err
		}
		uc.mu.Lock()
		if existing, ok := uc.cache[key]; ok {
			uc.mu.Unlock()
			return existing, nil
		}
		uc.cache[key] = cfg
		uc.mu.Unlock()
		return cfg, nil
	}
	uc.mu.Unlock()

	if err := uc.loadAll(ctx); err != nil {
		return nil, err
	}
	uc.mu.RLock()
	cfg, ok := uc.cache[key]
	uc.mu.RUnlock()
	if ok {
		return cfg, nil
	}
	return nil, entityError.ErrConfigNotFound
}

func (uc *DynamicConfigUseCase) GetAll(ctx context.Context) ([]*entity.Config, error) {
	uc.mu.RLock()
	if time.Since(uc.lastLoad) < configCacheTTL {
		list := make([]*entity.Config, 0, len(uc.cache))
		for _, cfg := range uc.cache {
			list = append(list, cfg)
		}
		uc.mu.RUnlock()
		return list, nil
	}
	uc.mu.RUnlock()

	uc.mu.Lock()
	if time.Since(uc.lastLoad) < configCacheTTL {
		list := make([]*entity.Config, 0, len(uc.cache))
		for _, cfg := range uc.cache {
			list = append(list, cfg)
		}
		uc.mu.Unlock()
		return list, nil
	}
	uc.mu.Unlock()

	if err := uc.loadAll(ctx); err != nil {
		return nil, err
	}
	uc.mu.RLock()
	list := make([]*entity.Config, 0, len(uc.cache))
	for _, cfg := range uc.cache {
		list = append(list, cfg)
	}
	uc.mu.RUnlock()
	return list, nil
}

func (uc *DynamicConfigUseCase) Set(ctx context.Context, key, value, description string, valueType entity.ConfigValueType, actorID uuid.UUID, clientIP string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("DynamicConfigUseCase - Set: config key is required")
	}
	if err := uc.validateValueType(valueType, value); err != nil {
		return err
	}
	cfg := &entity.Config{
		Key:         key,
		Value:       value,
		ValueType:   valueType,
		Description: description,
	}
	if err := uc.repo.Upsert(ctx, cfg); err != nil {
		return fmt.Errorf("DynamicConfigUseCase - Set: %w", err)
	}
	uc.invalidate()

	auditLog := &entity.AuditLog{
		UserID:     &actorID,
		Action:     entity.AuditActionUpdate,
		EntityType: "config",
		EntityID:   key,
		IP:         clientIP,
		Details: map[string]any{
			"message": "config updated",
			"key":     key,
		},
	}
	if err := uc.auditLogRepo.Create(ctx, auditLog); err != nil {
		return fmt.Errorf("DynamicConfigUseCase - Set - Create audit: %w", err)
	}
	return nil
}

func (uc *DynamicConfigUseCase) Delete(ctx context.Context, key string, actorID uuid.UUID, clientIP string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("DynamicConfigUseCase - Delete: config key is required")
	}
	if _, err := uc.repo.GetByKey(ctx, key); err != nil {
		return err
	}
	if err := uc.repo.Delete(ctx, key); err != nil {
		return fmt.Errorf("DynamicConfigUseCase - Delete: %w", err)
	}
	uc.invalidate()

	auditLog := &entity.AuditLog{
		UserID:     &actorID,
		Action:     entity.AuditActionDelete,
		EntityType: "config",
		EntityID:   key,
		IP:         clientIP,
		Details: map[string]any{
			"message": "config deleted",
			"key":     key,
		},
	}
	if err := uc.auditLogRepo.Create(ctx, auditLog); err != nil {
		return fmt.Errorf("DynamicConfigUseCase - Delete - Create audit: %w", err)
	}
	return nil
}

func (uc *DynamicConfigUseCase) validateValueType(valueType entity.ConfigValueType, value string) error {
	switch valueType {
	case entity.ConfigTypeInt:
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("DynamicConfigUseCase - validateValueType: value must be a valid integer for type int")
		}
	case entity.ConfigTypeBool:
		if value != "true" && value != "false" {
			return fmt.Errorf("DynamicConfigUseCase - validateValueType: value must be true or false for type bool")
		}
	case entity.ConfigTypeString, entity.ConfigTypeJSON:
	default:
		return fmt.Errorf("DynamicConfigUseCase - validateValueType: invalid value_type: %s", valueType)
	}
	return nil
}

func (uc *DynamicConfigUseCase) GetString(ctx context.Context, key, defaultVal string) string {
	cfg, err := uc.Get(ctx, key)
	if err != nil || cfg == nil {
		return defaultVal
	}
	return cfg.Value
}

func (uc *DynamicConfigUseCase) GetInt(ctx context.Context, key string, defaultVal int) int {
	cfg, err := uc.Get(ctx, key)
	if err != nil || cfg == nil {
		return defaultVal
	}
	val, err := strconv.Atoi(cfg.Value)
	if err != nil {
		return defaultVal
	}
	return val
}

func (uc *DynamicConfigUseCase) GetBool(ctx context.Context, key string, defaultVal bool) bool {
	cfg, err := uc.Get(ctx, key)
	if err != nil || cfg == nil {
		return defaultVal
	}
	return cfg.Value == "true"
}
