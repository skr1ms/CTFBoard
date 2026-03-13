package competition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

const (
	localTTL                  = 5 * time.Second
	redisTTL                  = 60 * time.Second
	invalidateTimeout         = 2 * time.Second
	configsCacheKey           = "configs:all"
	configsInvChannel         = "configs:inv"
	loadAllKey                = "competition_params:loadAll"
	competitionParamKeyMaxLen = 100
	subBackoffInitial         = 1 * time.Second
	subBackoffMax             = 30 * time.Second
)

var competitionParamKeyRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

var allowedCategories = map[string]struct{}{
	"general": {}, "theme": {}, "visibility": {}, "scoring": {},
	"email": {}, "social": {}, "legal": {}, "advanced": {},
}

func validateCategory(category string) error {
	if category == "" {
		return httperr.NewValidationErrorf("category is required")
	}
	if _, ok := allowedCategories[category]; !ok {
		return httperr.NewValidationErrorf("invalid category %q: must be one of general, theme, visibility, scoring, email, social, legal, advanced", category)
	}
	return nil
}

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
	Cache        cache.KeyValueStore
	PubSub       cache.PubSubStore
}

var _ usecase.CompetitionParamUseCase = (*CompetitionParamUseCase)(nil)

func NewCompetitionParamUseCase(deps CompetitionParamDeps) *CompetitionParamUseCase {
	if deps.Logger == nil {
		deps.Logger = logger.Noop()
	}
	uc := &CompetitionParamUseCase{
		deps:  deps,
		cache: make(map[string]*entity.CompetitionParam),
	}
	if uc.deps.PubSub != nil {
		go uc.subscribeInvalidation()
	}
	return uc
}

// invalidateLocal clears local cache and forgets singleflight key; concurrent ensureLoaded may start a new loadAll (accepted: stale window bounded by localTTL).
func (uc *CompetitionParamUseCase) invalidateLocal() {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.lastLoad = time.Time{}
	uc.sf.Forget(loadAllKey)
}

// invalidate clears local cache, deletes Redis cache, and publishes invalidation message; concurrent Get/GetAll use stale local cache.
func (uc *CompetitionParamUseCase) invalidate() {
	uc.invalidateLocal()
	ctx, cancel := context.WithTimeout(context.Background(), invalidateTimeout)
	defer cancel()
	if uc.deps.Cache != nil {
		if err := uc.deps.Cache.Del(ctx, configsCacheKey); err != nil {
			uc.deps.Logger.WithError(err).Warn("competition_params: cache invalidation failed", logger.Fields{"key": configsCacheKey})
		}
	}
	if uc.deps.PubSub != nil {
		if err := uc.deps.PubSub.Publish(ctx, configsInvChannel, "1"); err != nil {
			uc.deps.Logger.WithError(err).Warn("competition_params: pubsub invalidation broadcast failed", logger.Fields{"channel": configsInvChannel})
		}
	}
}

func (uc *CompetitionParamUseCase) subscribeInvalidation() {
	backoff := subBackoffInitial
	for {
		ctx := context.Background()
		ch, err := uc.deps.PubSub.Subscribe(ctx, configsInvChannel)
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("competition_params: subscribe to invalidation channel failed, retrying",
				logger.Fields{"backoff_sec": backoff.Seconds()})
			time.Sleep(backoff)
			if backoff < subBackoffMax {
				backoff *= 2
				if backoff > subBackoffMax {
					backoff = subBackoffMax
				}
			}
			continue
		}
		backoff = subBackoffInitial
		for msg := range ch {
			_ = msg
			uc.invalidateLocal()
		}
		uc.deps.Logger.Warn("competition_params: configs invalidation subscriber stopped, reconnecting",
			logger.Fields{"backoff_sec": backoff.Seconds()})
		time.Sleep(backoff)
		if backoff < subBackoffMax {
			backoff *= 2
			if backoff > subBackoffMax {
				backoff = subBackoffMax
			}
		}
	}
}

func (uc *CompetitionParamUseCase) loadFromRedis(ctx context.Context) error {
	if uc.deps.Cache == nil {
		return fmt.Errorf("no cache")
	}
	raw, err := uc.deps.Cache.Get(ctx, configsCacheKey)
	if err != nil {
		return err
	}
	var params []*entity.CompetitionParam
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return err
	}
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.cache = make(map[string]*entity.CompetitionParam, len(params))
	for _, p := range params {
		uc.cache[p.Key] = p
	}
	uc.lastLoad = time.Now()
	return nil
}

func (uc *CompetitionParamUseCase) loadAll(ctx context.Context) error {
	params, err := uc.deps.Repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("CompetitionParamUseCase - loadAll - CompetitionParamRepo.GetAll: %w", err)
	}
	uc.mu.Lock()
	uc.cache = make(map[string]*entity.CompetitionParam, len(params))
	for _, p := range params {
		uc.cache[p.Key] = p
	}
	uc.lastLoad = time.Now()
	uc.mu.Unlock()
	if uc.deps.Cache != nil {
		if b, err := json.Marshal(params); err == nil {
			_ = uc.deps.Cache.Set(ctx, configsCacheKey, b, redisTTL)
		}
	}
	return nil
}

func (uc *CompetitionParamUseCase) ensureLoaded(ctx context.Context) error {
	_, err, _ := uc.sf.Do(loadAllKey, func() (any, error) {
		if uc.deps.Cache != nil {
			if err := uc.loadFromRedis(ctx); err == nil {
				return nil, nil
			}
		}
		return nil, uc.loadAll(ctx)
	})
	return err
}

func (uc *CompetitionParamUseCase) Get(ctx context.Context, key string) (*entity.CompetitionParam, error) {
	key = strings.TrimSpace(key)
	if err := validateCompetitionParamKey(key); err != nil {
		return nil, err
	}
	uc.mu.RLock()
	cacheValid := time.Since(uc.lastLoad) < localTTL
	if cacheValid {
		if p, ok := uc.cache[key]; ok {
			uc.mu.RUnlock()
			return p, nil
		}
	}
	uc.mu.RUnlock()

	if !cacheValid {
		loadCtx := context.WithoutCancel(ctx)
		if err := uc.ensureLoaded(loadCtx); err != nil {
			return nil, fmt.Errorf("CompetitionParamUseCase - Get - ensureLoaded: %w", err)
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
			if errors.Is(err, httperr.ErrCompetitionParamNotFound) {
				if def, ok := entity.ConfigRegistry[key]; ok {
					return paramFromDef(def), nil
				}
			}
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

func paramFromDef(def entity.ConfigDef) *entity.CompetitionParam {
	return &entity.CompetitionParam{
		Key: def.Key, Value: def.DefaultValue, ValueType: def.ValueType,
		Category: def.Category, Description: def.Description,
	}
}

func (uc *CompetitionParamUseCase) GetAll(ctx context.Context) ([]*entity.CompetitionParam, error) {
	uc.mu.RLock()
	cacheValid := time.Since(uc.lastLoad) < localTTL
	uc.mu.RUnlock()
	if !cacheValid {
		loadCtx := context.WithoutCancel(ctx)
		if err := uc.ensureLoaded(loadCtx); err != nil {
			return nil, fmt.Errorf("CompetitionParamUseCase - GetAll - ensureLoaded: %w", err)
		}
	}
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	merged := make(map[string]*entity.CompetitionParam, len(entity.ConfigRegistry)+len(uc.cache))
	for k, def := range entity.ConfigRegistry {
		merged[k] = paramFromDef(def)
	}
	for k, p := range uc.cache {
		merged[k] = p
	}
	list := make([]*entity.CompetitionParam, 0, len(merged))
	for _, p := range merged {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Key < list[j].Key })
	return list, nil
}

func (uc *CompetitionParamUseCase) GetByCategory(ctx context.Context, category string) ([]*entity.CompetitionParam, error) {
	if err := validateCategory(category); err != nil {
		return nil, err
	}
	all, err := uc.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamUseCase - GetByCategory - GetAll: %w", err)
	}
	out := make([]*entity.CompetitionParam, 0)
	for _, p := range all {
		if p.Category == category {
			out = append(out, p)
		}
	}
	return out, nil
}

func (uc *CompetitionParamUseCase) SetBatch(ctx context.Context, params []*entity.CompetitionParam, actorID uuid.UUID, clientIP string) error {
	if len(params) == 0 {
		return nil
	}
	toUpsert := make([]*entity.CompetitionParam, len(params))
	for i, p := range params {
		cat := p.Category
		if cat == "" {
			if def, ok := entity.ConfigRegistry[p.Key]; ok {
				cat = def.Category
			} else {
				cat = "general"
			}
		}
		toUpsert[i] = &entity.CompetitionParam{
			Key: p.Key, Value: p.Value, ValueType: p.ValueType, Category: cat, Description: p.Description,
		}
	}
	keys := make([]string, 0, len(toUpsert))
	for _, p := range toUpsert {
		if err := validateCompetitionParamKey(p.Key); err != nil {
			return err
		}
		if err := uc.validateValueType(p.ValueType, p.Value); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - SetBatch - validateValueType key %q: %w", p.Key, err)
		}
		if err := validateCategory(p.Category); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - SetBatch - key %q: %w", p.Key, err)
		}
		keys = append(keys, p.Key)
	}
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		for _, p := range toUpsert {
			if err := uc.deps.Repo.Upsert(ctx, p); err != nil {
				return fmt.Errorf("CompetitionParamUseCase - SetBatch - CompetitionParamRepo.Upsert key %q: %w", p.Key, err)
			}
		}
		auditLog := &entity.AuditLog{
			UserID:     &actorID,
			Action:     entity.AuditActionUpdate,
			EntityType: "competition_param",
			EntityID:   "batch",
			IP:         clientIP,
			Details: map[string]any{
				"message": "competition params batch updated",
				"keys":    keys,
			},
		}
		if err := uc.deps.AuditLogRepo.Create(ctx, auditLog); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - SetBatch - AuditLogRepo.Create: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("CompetitionParamUseCase - SetBatch - TM.Run: %w", err)
	}
	uc.invalidate()
	return nil
}

func validateCompetitionParamKey(key string) error {
	if key == "" {
		return httperr.ErrCompetitionParamKeyRequired
	}
	if len(key) > competitionParamKeyMaxLen {
		return httperr.NewValidationErrorf("config key must be at most %d characters", competitionParamKeyMaxLen)
	}
	if !competitionParamKeyRe.MatchString(key) {
		return httperr.NewValidationErrorf("config key must contain only letters, digits, dots, underscores and hyphens")
	}
	return nil
}

func (uc *CompetitionParamUseCase) Set(ctx context.Context, key, value, description string, valueType entity.CompetitionParamValueType, actorID uuid.UUID, clientIP string) error {
	key = strings.TrimSpace(key)
	if err := validateCompetitionParamKey(key); err != nil {
		return err
	}
	if err := uc.validateValueType(valueType, value); err != nil {
		return fmt.Errorf("CompetitionParamUseCase - Set - validateValueType: %w", err)
	}
	category := "general"
	if def, ok := entity.ConfigRegistry[key]; ok {
		category = def.Category
	}
	p := &entity.CompetitionParam{
		Key:         key,
		Value:       value,
		ValueType:   valueType,
		Category:    category,
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
	if err := validateCompetitionParamKey(key); err != nil {
		return err
	}
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.Repo.GetByKeyForUpdate(ctx, key); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - Delete - CompetitionParamRepo.GetByKeyForUpdate: %w", err)
		}
		if err := uc.deps.Repo.Delete(ctx, key); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - Delete - CompetitionParamRepo.Delete: %w", err)
		}
		details := map[string]any{
			"message": "competition param deleted",
			"key":     key,
		}
		if _, inRegistry := entity.ConfigRegistry[key]; inRegistry {
			details["reset_to_default_from_registry"] = true
		}
		auditLog := &entity.AuditLog{
			UserID:     &actorID,
			Action:     entity.AuditActionDelete,
			EntityType: "competition_param",
			EntityID:   key,
			IP:         clientIP,
			Details:    details,
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
	case entity.CompetitionParamTypeString:
	case entity.CompetitionParamTypeJSON:
		if !json.Valid([]byte(value)) {
			return httperr.ErrCompetitionParamInvalidValueType
		}
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
