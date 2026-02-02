package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type AwardRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewAwardRepo(db *pgxpool.Pool) *AwardRepo {
	return &AwardRepo{db: db, q: sqlc.New(db)}
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

func (r *AwardRepo) Create(ctx context.Context, a *entity.Award) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	value, err := intToInt32Safe(a.Value)
	if err != nil {
		return fmt.Errorf("AwardRepo - Create: %w", err)
	}
	err = r.q.CreateAward(ctx, sqlc.CreateAwardParams{
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

func (r *AwardRepo) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error) {
	rows, err := r.q.GetAwardsByTeamID(ctx, teamID)
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
	total, err := r.q.GetTeamTotalAwards(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("AwardRepo - GetTeamTotalAwards: %w", err)
	}
	return int(total), nil
}

func (r *AwardRepo) GetAll(ctx context.Context) ([]*entity.Award, error) {
	rows, err := r.q.GetAllAwards(ctx)
	if err != nil {
		return nil, fmt.Errorf("AwardRepo - GetAll: %w", err)
	}
	out := make([]*entity.Award, 0, len(rows))
	for _, a := range rows {
		out = append(out, toEntityAward(a))
	}
	return out, nil
}
