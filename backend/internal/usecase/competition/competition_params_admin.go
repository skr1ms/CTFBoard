package competition

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

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
		cat := domain.ConfigCategoryGeneral

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

	uc.invalidate(ctx)

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

	cat := domain.ConfigCategoryGeneral
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

	uc.invalidate(ctx)

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

	uc.invalidate(ctx)

	return nil
}
