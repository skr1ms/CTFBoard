package competition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const (
	localTTL                  = 5 * time.Second
	redisTTL                  = 60 * time.Second
	negativeCacheTTL          = 30 * time.Second
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
	deps          CompetitionParamDeps
	cache         map[string]*domain.CompetitionParam
	negativeCache map[string]time.Time
	mu            sync.RWMutex
	lastLoad      time.Time
	sf            singleflight.Group
}

type CompetitionParamDeps struct {
	Repo         repo.CompetitionParamRepository
	AuditLogRepo repo.AuditLogRepository
	TM           repo.TransactionManager
	Logger       logkit.Logger
	Cache        cachekit.KeyValueStore
	PubSub       cachekit.PubSubStore
	StopContext  context.Context
}

var _ usecase.CompetitionParamUseCase = (*CompetitionParamUseCase)(nil)

func NewCompetitionParamUseCase(ctx context.Context, deps CompetitionParamDeps) *CompetitionParamUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}
	if deps.StopContext == nil {
		deps.StopContext = ctx
	}
	uc := &CompetitionParamUseCase{
		deps:          deps,
		cache:         make(map[string]*domain.CompetitionParam),
		negativeCache: make(map[string]time.Time),
	}
	if uc.deps.PubSub != nil {
		stopCtx := uc.deps.StopContext
		if stopCtx == nil {
			stopCtx = context.Background()
		}
		go uc.subscribeInvalidation(stopCtx)
	}
	return uc
}

// invalidateLocal clears local cache, negative cache, and forgets singleflight key; concurrent ensureLoaded may start a new loadAll (accepted: stale window bounded by localTTL).
func (uc *CompetitionParamUseCase) invalidateLocal() {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.lastLoad = time.Time{}
	uc.negativeCache = make(map[string]time.Time)
	uc.sf.Forget(loadAllKey)
}

// invalidate clears local cache, deletes Redis cache, and publishes invalidation message; concurrent Get/GetAll use stale local cache.
func (uc *CompetitionParamUseCase) invalidate() {
	uc.invalidateLocal()
	ctx, cancel := context.WithTimeout(context.Background(), invalidateTimeout)
	defer cancel()
	if uc.deps.Cache != nil {
		if err := uc.deps.Cache.Del(ctx, configsCacheKey); err != nil {
			uc.deps.Logger.WithError(err).Warn("competition_params: cache invalidation failed", logkit.Fields{"key": configsCacheKey})
		}
	}
	if uc.deps.PubSub != nil {
		if err := uc.deps.PubSub.Publish(ctx, configsInvChannel, "1"); err != nil {
			uc.deps.Logger.WithError(err).Warn("competition_params: pubsub invalidation broadcast failed", logkit.Fields{"channel": configsInvChannel})
		}
	}
}

func (uc *CompetitionParamUseCase) subscribeInvalidation(stopCtx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = subBackoffInitial
	bo.MaxInterval = subBackoffMax
	bo.MaxElapsedTime = 0
	notify := func(err error, d time.Duration) {
		uc.deps.Logger.WithError(err).Warn("competition_params: subscribe to invalidation channel failed, retrying",
			logkit.Fields{"backoff_sec": d.Seconds()})
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
			ch, err = uc.deps.PubSub.Subscribe(stopCtx, configsInvChannel)
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
			case msg, ok := <-ch:
				if !ok {
					break readLoop
				}
				_ = msg
				uc.invalidateLocal()
			}
		}
		next := bo.NextBackOff()
		if next == backoff.Stop {
			return
		}
		uc.deps.Logger.Warn("competition_params: configs invalidation subscriber stopped, reconnecting",
			logkit.Fields{"backoff_sec": next.Seconds()})
		t := time.NewTimer(next)
		select {
		case <-stopCtx.Done():
			t.Stop()
			return
		case <-t.C:
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
	var params []*domain.CompetitionParam
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return err
	}
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.cache = make(map[string]*domain.CompetitionParam, len(params))
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
	uc.cache = make(map[string]*domain.CompetitionParam, len(params))
	for _, p := range params {
		uc.cache[p.Key] = p
	}
	uc.lastLoad = time.Now()
	uc.mu.Unlock()
	if uc.deps.Cache != nil {
		if b, err := json.Marshal(params); err == nil {
			if setErr := uc.deps.Cache.Set(ctx, configsCacheKey, b, redisTTL); setErr != nil {
				uc.deps.Logger.WithError(setErr).Warn("competition_params: redis cache set failed", logkit.Fields{"key": configsCacheKey})
			}
		}
	}
	return nil
}

func (uc *CompetitionParamUseCase) ensureLoaded(ctx context.Context) error {
	_, err, _ := uc.sf.Do(loadAllKey, func() (any, error) {
		uc.mu.RLock()
		cacheValid := time.Since(uc.lastLoad) < localTTL
		uc.mu.RUnlock()
		if cacheValid {
			return nil, nil
		}
		if uc.deps.Cache != nil {
			if err := uc.loadFromRedis(ctx); err == nil {
				return nil, nil
			}
		}
		return nil, uc.loadAll(ctx)
	})
	return err
}

func (uc *CompetitionParamUseCase) Get(ctx context.Context, key string) (*domain.CompetitionParam, error) {
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
	if !ok {
		if exp, ok := uc.negativeCache[key]; ok && time.Now().Before(exp) {
			uc.mu.RUnlock()
			return nil, httperr.ErrCompetitionParamNotFound
		}
	}
	uc.mu.RUnlock()
	if ok {
		return p, nil
	}

	sfKey := "competition_params:key:" + key
	v, err, _ := uc.sf.Do(sfKey, func() (any, error) {
		c, err := uc.deps.Repo.GetByKey(context.WithoutCancel(ctx), key)
		if err != nil {
			if errors.Is(err, httperr.ErrCompetitionParamNotFound) {
				if def, ok := domain.GetConfigDef(key); ok {
					p := paramFromDef(def)
					uc.mu.Lock()
					uc.cache[key] = p
					uc.mu.Unlock()
					return p, nil
				}
				uc.mu.Lock()
				uc.negativeCache[key] = time.Now().Add(negativeCacheTTL)
				uc.mu.Unlock()
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
	c, ok := v.(*domain.CompetitionParam)
	if !ok {
		return nil, fmt.Errorf("CompetitionParamUseCase - Get: unexpected cache type for key %q", key)
	}
	return c, nil
}

func paramFromDef(def domain.ConfigDef) *domain.CompetitionParam {
	return &domain.CompetitionParam{
		Key: def.Key, Value: def.DefaultValue, ValueType: def.ValueType,
		Category: def.Category, Description: def.Description,
	}
}

func (uc *CompetitionParamUseCase) GetAll(ctx context.Context) ([]*domain.CompetitionParam, error) {
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
	merged := make(map[string]*domain.CompetitionParam, domain.ConfigRegistryCount()+len(uc.cache))
	domain.RangeConfigRegistry(func(k string, def domain.ConfigDef) bool {
		merged[k] = paramFromDef(def)
		return true
	})
	for k, p := range uc.cache {
		merged[k] = p
	}
	list := make([]*domain.CompetitionParam, 0, len(merged))
	for _, p := range merged {
		list = append(list, p)
	}
	slices.SortFunc(list, func(a, b *domain.CompetitionParam) int { return strings.Compare(a.Key, b.Key) })
	return list, nil
}

func (uc *CompetitionParamUseCase) GetByCategory(ctx context.Context, category string) ([]*domain.CompetitionParam, error) {
	if err := validateCategory(category); err != nil {
		return nil, err
	}
	merged := make(map[string]*domain.CompetitionParam)
	domain.RangeConfigRegistry(func(k string, def domain.ConfigDef) bool {
		if def.Category == category {
			merged[k] = paramFromDef(def)
		}
		return true
	})
	dbParams, err := uc.deps.Repo.GetByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamUseCase - GetByCategory - Repo.GetByCategory: %w", err)
	}
	for _, p := range dbParams {
		merged[p.Key] = p
	}
	out := make([]*domain.CompetitionParam, 0, len(merged))
	for _, p := range merged {
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b *domain.CompetitionParam) int { return strings.Compare(a.Key, b.Key) })
	return out, nil
}

func (uc *CompetitionParamUseCase) SetBatch(ctx context.Context, params []*domain.CompetitionParam, actorID uuid.UUID, clientIP string) error {
	if len(params) == 0 {
		return nil
	}
	toUpsert := make([]*domain.CompetitionParam, len(params))
	for i, p := range params {
		key := strings.TrimSpace(p.Key)
		cat := "general"
		vt := p.ValueType
		if def, ok := domain.GetConfigDef(key); ok {
			cat = def.Category
			vt = def.ValueType
		} else if p.Category != "" {
			cat = p.Category
		}
		toUpsert[i] = &domain.CompetitionParam{
			Key: key, Value: p.Value, ValueType: vt, Category: cat, Description: p.Description,
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
		auditLog := &domain.AuditLog{
			UserID:     &actorID,
			Action:     domain.AuditActionUpdate,
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

func (uc *CompetitionParamUseCase) Set(ctx context.Context, key, value, description string, valueType domain.CompetitionParamValueType, category string, actorID uuid.UUID, clientIP string) error {
	key = strings.TrimSpace(key)
	if err := validateCompetitionParamKey(key); err != nil {
		return err
	}
	cat := "general"
	vt := valueType
	if def, ok := domain.GetConfigDef(key); ok {
		cat = def.Category
		vt = def.ValueType
	} else if category != "" {
		if err := validateCategory(category); err != nil {
			return err
		}
		cat = category
	}
	if err := uc.validateValueType(vt, value); err != nil {
		return fmt.Errorf("CompetitionParamUseCase - Set - validateValueType: %w", err)
	}
	p := &domain.CompetitionParam{
		Key:         key,
		Value:       value,
		ValueType:   vt,
		Category:    cat,
		Description: description,
	}
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.Repo.Upsert(ctx, p); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - Set - CompetitionParamRepo.Upsert: %w", err)
		}
		auditLog := &domain.AuditLog{
			UserID:     &actorID,
			Action:     domain.AuditActionUpdate,
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
		if _, inRegistry := domain.GetConfigDef(key); inRegistry {
			details["reset_to_default_from_registry"] = true
		}
		auditLog := &domain.AuditLog{
			UserID:     &actorID,
			Action:     domain.AuditActionDelete,
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

func (uc *CompetitionParamUseCase) validateValueType(valueType domain.CompetitionParamValueType, value string) error {
	switch valueType {
	case domain.CompetitionParamTypeInt:
		if _, err := strconv.Atoi(value); err != nil {
			return httperr.ErrCompetitionParamInvalidValueType
		}
	case domain.CompetitionParamTypeBool:
		if value != "true" && value != "false" {
			return httperr.ErrCompetitionParamInvalidValueType
		}
	case domain.CompetitionParamTypeString:
	case domain.CompetitionParamTypeJSON:
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
		uc.deps.Logger.WithError(err).Warn("competition_params: GetString failed, returning default", logkit.Fields{"key": key})
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
		uc.deps.Logger.WithError(err).Warn("competition_params: GetInt failed, returning default", logkit.Fields{"key": key})
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
		uc.deps.Logger.WithError(err).Warn("competition_params: GetBool failed, returning default", logkit.Fields{"key": key})
		return defaultVal
	}
	if p == nil {
		return defaultVal
	}
	return p.Value == "true"
}
