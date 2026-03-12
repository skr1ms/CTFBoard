package persistent

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type CompetitionParamRepo struct {
	pool *pgxpool.Pool
}

var _ repo.CompetitionParamRepository = (*CompetitionParamRepo)(nil)

func NewCompetitionParamRepo(pool *pgxpool.Pool) *CompetitionParamRepo {
	return &CompetitionParamRepo{pool: pool}
}

func (r *CompetitionParamRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func toEntityCompetitionParam(row sqlc.CompetitionParam) *entity.CompetitionParam {
	return &entity.CompetitionParam{
		Key:         row.Key,
		Value:       row.Value,
		ValueType:   entity.CompetitionParamValueType(row.ValueType),
		Description: ptrStrToStr(row.Description),
		UpdatedAt:   ptrTimeToTime(timestamptzToTime(row.UpdatedAt)),
	}
}

func (r *CompetitionParamRepo) GetAll(ctx context.Context) ([]*entity.CompetitionParam, error) {
	rows, err := r.q(ctx).GetAllConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamRepo - GetAll: %w", err)
	}
	out := make([]*entity.CompetitionParam, len(rows))
	for i := range rows {
		out[i] = toEntityCompetitionParam(rows[i])
	}
	return out, nil
}

func (r *CompetitionParamRepo) GetByKey(ctx context.Context, key string) (*entity.CompetitionParam, error) {
	row, err := r.q(ctx).GetConfigByKey(ctx, key)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrCompetitionParamNotFound
		}
		return nil, fmt.Errorf("CompetitionParamRepo - GetByKey: %w", err)
	}
	return toEntityCompetitionParam(row), nil
}

func (r *CompetitionParamRepo) GetByKeyForUpdate(ctx context.Context, key string) (*entity.CompetitionParam, error) {
	row, err := r.q(ctx).GetConfigByKeyForUpdate(ctx, key)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrCompetitionParamNotFound
		}
		return nil, fmt.Errorf("CompetitionParamRepo - GetByKeyForUpdate: %w", err)
	}
	return toEntityCompetitionParam(row), nil
}

func (r *CompetitionParamRepo) Upsert(ctx context.Context, p *entity.CompetitionParam) error {
	desc := strPtrOrNil(p.Description)
	err := r.q(ctx).UpsertConfig(ctx, sqlc.UpsertConfigParams{
		Key:         p.Key,
		Value:       p.Value,
		ValueType:   string(p.ValueType),
		Description: desc,
	})
	if err != nil {
		return fmt.Errorf("CompetitionParamRepo - Upsert: %w", err)
	}
	return nil
}

func (r *CompetitionParamRepo) Delete(ctx context.Context, key string) error {
	if err := r.q(ctx).DeleteConfig(ctx, key); err != nil {
		return fmt.Errorf("CompetitionParamRepo - Delete: %w", err)
	}
	return nil
}
