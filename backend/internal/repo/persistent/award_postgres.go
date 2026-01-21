package persistent

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AwardRepo struct {
	pool *pgxpool.Pool
}

func NewAwardRepo(pool *pgxpool.Pool) *AwardRepo {
	return &AwardRepo{pool: pool}
}

func (r *AwardRepo) GetTeamTotalAwards(ctx context.Context, teamId uuid.UUID) (int, error) {
	query := squirrel.Select("COALESCE(SUM(value), 0)").
		From("awards").
		Where(squirrel.Eq{"team_id": teamId}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("AwardRepo - GetTeamTotalAwards - BuildQuery: %w", err)
	}

	var total int
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("AwardRepo - GetTeamTotalAwards - Scan: %w", err)
	}

	return total, nil
}
