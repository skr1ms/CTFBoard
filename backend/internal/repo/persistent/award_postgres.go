package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AwardRepo struct {
	pool *pgxpool.Pool
}

var _ repo.AwardRepository = (*AwardRepo)(nil)

func NewAwardRepo(pool *pgxpool.Pool) *AwardRepo {
	return &AwardRepo{pool: pool}
}

func (r *AwardRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func toEntityAward(a sqlc.Award) *entity.Award {
	return &entity.Award{
		ID:          a.ID,
		TeamID:      a.TeamID,
		Value:       int(a.Value),
		Description: a.Description,
		CreatedBy:   a.CreatedBy,
		CreatedAt:   ptrTimeToTime(a.CreatedAt),
	}
}

func (r *AwardRepo) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error) {
	rows, err := r.q(ctx).GetAwardsByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("AwardRepo - GetByTeamID: %w", err)
	}
	out := make([]*entity.Award, 0, len(rows))
	for _, a := range rows {
		out = append(out, toEntityAward(a))
	}
	return out, nil
}

func (r *AwardRepo) GetTeamTotalAwards(ctx context.Context, teamID uuid.UUID) (int, error) {
	total, err := r.q(ctx).GetTeamTotalAwards(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("AwardRepo - GetTeamTotalAwards: %w", err)
	}
	return int(total), nil
}

func (r *AwardRepo) GetAll(ctx context.Context) ([]*entity.Award, error) {
	rows, err := r.q(ctx).GetAllAwards(ctx)
	if err != nil {
		return nil, fmt.Errorf("AwardRepo - GetAll: %w", err)
	}
	out := make([]*entity.Award, 0, len(rows))
	for _, a := range rows {
		out = append(out, toEntityAward(a))
	}
	return out, nil
}

func (r *AwardRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Award, error) {
	a, err := r.q(ctx).GetAwardByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrAwardNotFound
		}
		return nil, fmt.Errorf("AwardRepo - GetByID: %w", err)
	}
	return toEntityAward(a), nil
}

func (r *AwardRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := r.q(ctx).DeleteAward(ctx, ID); err != nil {
		return fmt.Errorf("AwardRepo - Delete: %w", err)
	}
	return nil
}

func (r *AwardRepo) DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error {
	if err := r.q(ctx).DeleteAwardsByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("AwardRepo - DeleteByTeamID: %w", err)
	}
	return nil
}

func (r *AwardRepo) Create(ctx context.Context, a *entity.Award) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	value, err := intToInt32Safe(a.Value)
	if err != nil {
		return fmt.Errorf("AwardRepo - Create: %w", err)
	}
	err = r.q(ctx).CreateAward(ctx, sqlc.CreateAwardParams{
		ID:          a.ID,
		TeamID:      a.TeamID,
		Value:       value,
		Description: a.Description,
		CreatedBy:   a.CreatedBy,
		CreatedAt:   &a.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("AwardRepo - Create: %w", err)
	}
	return nil
}
