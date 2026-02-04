package persistent

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type ConfigRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewConfigRepo(pool *pgxpool.Pool) *ConfigRepo {
	return &ConfigRepo{pool: pool, q: sqlc.New(pool)}
}

func configRowToEntity(row sqlc.Config) *entity.Config {
	return &entity.Config{
		Key:         row.Key,
		Value:       row.Value,
		ValueType:   entity.ConfigValueType(row.ValueType),
		Description: ptrStrToStr(row.Description),
		UpdatedAt:   ptrTimeToTime(row.UpdatedAt),
	}
}

func (r *ConfigRepo) GetAll(ctx context.Context) ([]*entity.Config, error) {
	rows, err := r.q.GetAllConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("ConfigRepo - GetAll: %w", err)
	}
	out := make([]*entity.Config, len(rows))
	for i := range rows {
		out[i] = configRowToEntity(rows[i])
	}
	return out, nil
}

func (r *ConfigRepo) GetByKey(ctx context.Context, key string) (*entity.Config, error) {
	row, err := r.q.GetConfigByKey(ctx, key)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrConfigNotFound
		}
		return nil, fmt.Errorf("ConfigRepo - GetByKey: %w", err)
	}
	return configRowToEntity(row), nil
}

func (r *ConfigRepo) Upsert(ctx context.Context, cfg *entity.Config) error {
	desc := strPtrOrNil(cfg.Description)
	err := r.q.UpsertConfig(ctx, sqlc.UpsertConfigParams{
		Key:         cfg.Key,
		Value:       cfg.Value,
		ValueType:   string(cfg.ValueType),
		Description: desc,
	})
	if err != nil {
		return fmt.Errorf("ConfigRepo - Upsert: %w", err)
	}
	return nil
}

func (r *ConfigRepo) Delete(ctx context.Context, key string) error {
	return r.q.DeleteConfig(ctx, key)
}
