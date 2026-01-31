package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
)

type AwardRepo struct {
	pool *pgxpool.Pool
}

func NewAwardRepo(pool *pgxpool.Pool) *AwardRepo {
	return &AwardRepo{pool: pool}
}

func (r *AwardRepo) GetTeamTotalAwards(ctx context.Context, teamID uuid.UUID) (int, error) {
	query := squirrel.Select("COALESCE(SUM(value), 0)").
		From("awards").
		Where(squirrel.Eq{"team_id": teamID}).
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

func (r *AwardRepo) Create(ctx context.Context, a *entity.Award) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()

	query := squirrel.Insert("awards").
		Columns("id", "team_id", "value", "description", "created_by", "created_at").
		Values(a.ID, a.TeamID, a.Value, a.Description, a.CreatedBy, a.CreatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("AwardRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("AwardRepo - Create - Exec: %w", err)
	}

	return nil
}

func (r *AwardRepo) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error) {
	query := squirrel.Select("id", "team_id", "value", "description", "created_by", "created_at").
		From("awards").
		Where(squirrel.Eq{"team_id": teamID}).
		OrderBy("created_at DESC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("AwardRepo - GetByTeamID - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("AwardRepo - GetByTeamID - Query: %w", err)
	}
	defer rows.Close()

	var awards []*entity.Award
	for rows.Next() {
		var a entity.Award
		err := rows.Scan(
			&a.ID,
			&a.TeamID,
			&a.Value,
			&a.Description,
			&a.CreatedBy,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("AwardRepo - GetByTeamID - Scan: %w", err)
		}
		awards = append(awards, &a)
	}

	return awards, nil
}
