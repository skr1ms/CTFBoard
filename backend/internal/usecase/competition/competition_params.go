package competition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
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
	localTTL                  = 5 * time.Second
	redisTTL                  = 60 * time.Second
	negativeCacheTTL          = 30 * time.Second
	invalidateTimeout         = 2 * time.Second
	configsCacheKey           = "configs:all"
	configsInvChannel         = "configs:inv"
	loadAllKey                = "competition_params:loadAll"
	competitionParamKeyMaxLen = 100
)

var competitionParamKeyRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

var errCacheNotInitialized = errors.New("cache not initialized")

var publicConfigKeys = []string{
	"ctf_name", "ctf_description", "ctf_logo", "tos_url", "privacy_url",
	"theme_color_primary", "theme_color_secondary", "theme_header_html", "theme_footer_html", "theme_dark_mode",
	"social_github", "social_discord", "social_twitter", "social_website",
	"challenge_visibility", "score_visibility", "account_visibility", "registration_visibility",
	"setup_complete", "email_verification_required", "timezone",
}

var allowedCategories = map[string]struct{}{
	"general": {}, "theme": {}, "visibility": {}, "scoring": {},
	"email": {}, "social": {}, "legal": {}, "advanced": {},
}

var allowedVisibilityValues = map[string][]string{
	"challenge_visibility":    {"public", "private", "hidden", "admins"},
	"score_visibility":        {"public", "private", "hidden", "admins", "admins_only"},
	"account_visibility":      {"public", "private", "hidden", "admins"},
	"registration_visibility": {"public", "private"},
}

func validateCategory(category string) error {
	if category == "" {
		return apperr.NewValidationErrorf("category is required")
	}

	if _, ok := allowedCategories[category]; !ok {
		return apperr.NewValidationErrorf("invalid category %q: must be one of general, theme, visibility, scoring, email, social, legal, advanced", category)
	}

	return nil
}

func validateRegisteredParamValue(key, value string) error {
	allowed, ok := allowedVisibilityValues[key]
	if !ok {
		return nil
	}

	if slices.Contains(allowed, value) {
		return nil
	}

	return apperr.NewValidationErrorf("invalid %s %q: must be one of %s", key, value, strings.Join(allowed, ", "))
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

// NewCompetitionParamUseCase initializes a CompetitionParamUseCase with an in-process map
// cache and a negative-miss cache. If PubSub is configured, a background goroutine is
// started to subscribe to Redis invalidation events and call invalidateLocal on each message.
func NewCompetitionParamUseCase(deps CompetitionParamDeps) *CompetitionParamUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	if deps.StopContext == nil {
		deps.StopContext = context.Background()
	}

	uc := &CompetitionParamUseCase{
		deps:          deps,
		cache:         make(map[string]*domain.CompetitionParam),
		negativeCache: make(map[string]time.Time),
	}
	if uc.deps.PubSub != nil {
		go cache.SubscribeInvalidation(uc.deps.StopContext, uc.deps.PubSub, configsInvChannel, uc.invalidateLocal, uc.deps.Logger, "competition_params")
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

// loadFromRedis attempts to populate the local in-memory cache from the Redis
// JSON blob stored under configsCacheKey. Returns an error when Redis is unavailable
// or the key is missing/stale, causing ensureLoaded to fall through to loadAll.
func (uc *CompetitionParamUseCase) loadFromRedis(ctx context.Context) error {
	if uc.deps.Cache == nil {
		return errCacheNotInitialized
	}

	raw, err := uc.deps.Cache.Get(ctx, configsCacheKey)
	if err != nil {
		return fmt.Errorf("CompetitionParamUseCase - loadFromRedis - Cache.Get: %w", err)
	}

	var params []*domain.CompetitionParam
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return fmt.Errorf("CompetitionParamUseCase - loadFromRedis - json.Unmarshal: %w", err)
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

// loadAll fetches all competition params from the database, rebuilds the
// in-memory map, and writes the result to Redis for other instances to share.
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

// ensureLoaded deduplicates concurrent cache misses with singleflight.
// Order of fallback: in-memory TTL check -> Redis -> database (loadAll).
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

		if err := uc.loadAll(ctx); err != nil {
			return nil, fmt.Errorf("CompetitionParamUseCase - ensureLoaded - loadAll: %w", err)
		}

		return nil, nil
	})

	return err
}

// Get returns a competition parameter by key using a multi-layer cache
// in-memory map (guarded by a read mutex, valid for localTTL) -> Redis -> database
// Concurrent cache misses are deduplicated with singleflight so only one
// database round-trip is made per key per miss window. When the database
// returns not-found, the registered config registry is checked; if the key has
// a default there, a synthetic param is returned and stored in the local cache
// Keys that are absent from both the database and the registry are stored in a
// negative cache for negativeCacheTTL to prevent repeated database queries.
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

			return nil, apperr.ErrCompetitionParamNotFound
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
			if errors.Is(err, apperr.ErrCompetitionParamNotFound) {
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

// GetAll returns all competition params merged with config registry defaults.
// Registry defaults are overridden by any database-persisted values. The result
// is sorted by key for deterministic output.
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

	maps.Copy(merged, uc.cache)

	list := make([]*domain.CompetitionParam, 0, len(merged))
	for _, p := range merged {
		list = append(list, p)
	}

	slices.SortFunc(list, func(a, b *domain.CompetitionParam) int { return strings.Compare(a.Key, b.Key) })

	return list, nil
}

// GetByCategory returns all competition parameters for a specific category,
// merging config registry defaults with any database-persisted values.
// Registry defaults for the category are loaded first; persisted values
// then override them so callers always see the effective configuration.
// The result is sorted by key for deterministic output.
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

// SetBatch validates and upserts a batch of competition parameters in a single
// transaction. For each entry it resolves the category and value type from the
// config registry when the key is registered, then validates the key format,
// value type, and category. All upserts run inside one transaction together
// with an audit log entry listing the affected keys. After the transaction
// commits, invalidate broadcasts a cache invalidation message over PubSub and
// clears both the local in-memory map and the Redis entry so all instances
// pick up the new values on their next request.
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

		if err := validateRegisteredParamValue(p.Key, p.Value); err != nil {
			return fmt.Errorf("CompetitionParamUseCase - SetBatch - key %q: %w", p.Key, err)
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
		return apperr.ErrCompetitionParamKeyRequired
	}

	if len(key) > competitionParamKeyMaxLen {
		return apperr.NewValidationErrorf("config key must be at most %d characters", competitionParamKeyMaxLen)
	}

	if !competitionParamKeyRe.MatchString(key) {
		return apperr.NewValidationErrorf("config key must contain only letters, digits, dots, underscores and hyphens")
	}

	return nil
}

// Set validates and upserts a single competition parameter inside a transaction.
// The key is trimmed and validated against the format rules. Category and value
// type are resolved from the config registry when the key is registered there;
// otherwise the caller-supplied values are used. After the transaction commits,
// invalidate broadcasts a cache-bust message over PubSub and clears both the
// local in-memory map and the Redis entry.
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

	if err := validateRegisteredParamValue(key, value); err != nil {
		return fmt.Errorf("CompetitionParamUseCase - Set - validateRegisteredParamValue: %w", err)
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

// Delete removes a persisted competition parameter inside a transaction.
// GetByKeyForUpdate acquires a pessimistic row lock to prevent concurrent
// deletes. The audit log entry notes whether the key has a registry default
// (effectively resetting the parameter to its default) or is being fully
// removed. After commit, invalidate clears all cache layers.
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

// GetPublic returns competition parameters for the public allow-list, falling
// back to registry defaults for keys that have not been persisted yet.
func (uc *CompetitionParamUseCase) GetPublic(ctx context.Context) ([]*domain.CompetitionParam, error) {
	all, err := uc.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamUseCase - GetPublic - GetAll: %w", err)
	}

	byKey := make(map[string]*domain.CompetitionParam, len(all))
	for _, p := range all {
		byKey[p.Key] = p
	}

	list := make([]*domain.CompetitionParam, 0, len(publicConfigKeys))
	for _, key := range publicConfigKeys {
		if p, ok := byKey[key]; ok {
			list = append(list, p)

			continue
		}

		if def, ok := domain.GetConfigDef(key); ok {
			list = append(list, paramFromDef(def))
		}
	}

	return list, nil
}

// validateValueType checks that value is parseable as the declared type:
// int -> strconv.Atoi, bool -> "true"/"false", string -> always valid,
// json -> json.Valid. Any unknown type also returns ErrCompetitionParamInvalidValueType.
func (uc *CompetitionParamUseCase) validateValueType(valueType domain.CompetitionParamValueType, value string) error {
	switch valueType {
	case domain.CompetitionParamTypeInt:
		if _, err := strconv.Atoi(value); err != nil {
			return apperr.ErrCompetitionParamInvalidValueType
		}
	case domain.CompetitionParamTypeBool:
		if value != "true" && value != "false" {
			return apperr.ErrCompetitionParamInvalidValueType
		}
	case domain.CompetitionParamTypeString:
	case domain.CompetitionParamTypeJSON:
		if !json.Valid([]byte(value)) {
			return apperr.ErrCompetitionParamInvalidValueType
		}
	default:
		return apperr.ErrCompetitionParamInvalidValueType
	}

	return nil
}

func (uc *CompetitionParamUseCase) GetString(ctx context.Context, key, defaultVal string) string {
	p, err := uc.Get(ctx, key)
	if err != nil {
		if errors.Is(err, apperr.ErrCompetitionParamNotFound) {
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
		if errors.Is(err, apperr.ErrCompetitionParamNotFound) {
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
		if errors.Is(err, apperr.ErrCompetitionParamNotFound) {
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
