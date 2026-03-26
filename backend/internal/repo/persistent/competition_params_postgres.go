package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type CompetitionParamRepo struct {
	BaseRepo
}

var _ repo.CompetitionParamRepository = (*CompetitionParamRepo)(nil)

func NewCompetitionParamRepo(pool *pgxpool.Pool) *CompetitionParamRepo {
	return &CompetitionParamRepo{BaseRepo: BaseRepo{pool: pool}}
}

func toDomainCompetitionParam(row sqlc.CompetitionParam) *domain.CompetitionParam {
	return &domain.CompetitionParam{
		Key:         row.Key,
		Value:       row.Value,
		ValueType:   domain.CompetitionParamValueType(row.ValueType),
		Description: lo.FromPtr(row.Description),
		Category:    row.Category,
		UpdatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt)),
	}
}

func (r *CompetitionParamRepo) GetAll(ctx context.Context) ([]*domain.CompetitionParam, error) {
	rows, err := r.Q(ctx).GetAllConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamRepo - GetAll: %w", err)
	}

	out := make([]*domain.CompetitionParam, len(rows))
	for i := range rows {
		out[i] = toDomainCompetitionParam(rows[i])
	}

	return out, nil
}

func (r *CompetitionParamRepo) GetByCategory(ctx context.Context, category string) ([]*domain.CompetitionParam, error) {
	rows, err := r.Q(ctx).GetConfigsByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamRepo - GetByCategory: %w", err)
	}

	out := make([]*domain.CompetitionParam, len(rows))
	for i := range rows {
		out[i] = toDomainCompetitionParam(rows[i])
	}

	return out, nil
}

func (r *CompetitionParamRepo) GetByKey(ctx context.Context, key string) (*domain.CompetitionParam, error) {
	row, err := r.Q(ctx).GetConfigByKey(ctx, key)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrCompetitionParamNotFound
		}

		return nil, fmt.Errorf("CompetitionParamRepo - GetByKey: %w", err)
	}

	return toDomainCompetitionParam(row), nil
}

func (r *CompetitionParamRepo) GetByKeyForUpdate(ctx context.Context, key string) (*domain.CompetitionParam, error) {
	row, err := r.Q(ctx).GetConfigByKeyForUpdate(ctx, key)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrCompetitionParamNotFound
		}

		return nil, fmt.Errorf("CompetitionParamRepo - GetByKeyForUpdate: %w", err)
	}

	return toDomainCompetitionParam(row), nil
}

func (r *CompetitionParamRepo) Upsert(ctx context.Context, p *domain.CompetitionParam) error {
	desc := lo.EmptyableToPtr(p.Description)
	now := time.Now()

	err := r.Q(ctx).UpsertConfig(ctx, sqlc.UpsertConfigParams{
		Key:         p.Key,
		Value:       p.Value,
		ValueType:   string(p.ValueType),
		Description: desc,
		Category:    p.Category,
		UpdatedAt:   pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		return fmt.Errorf("CompetitionParamRepo - Upsert: %w", err)
	}

	return nil
}

func (r *CompetitionParamRepo) Delete(ctx context.Context, key string) error {
	err := r.Q(ctx).DeleteConfig(ctx, key)
	if err != nil {
		return fmt.Errorf("CompetitionParamRepo - Delete: %w", err)
	}

	return nil
}
