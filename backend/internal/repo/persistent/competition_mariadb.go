package persistent

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
)

type CompetitionRepo struct {
	db *sql.DB
}

func NewCompetitionRepo(db *sql.DB) *CompetitionRepo {
	return &CompetitionRepo{db: db}
}

func (r *CompetitionRepo) Get(ctx context.Context) (*entity.Competition, error) {
	query := squirrel.Select("id", "name", "start_time", "end_time", "freeze_time", "is_paused", "is_public", "created_at", "updated_at").
		From("competition").
		Where(squirrel.Eq{"id": 1})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("CompetitionRepo - Get - BuildQuery: %w", err)
	}

	var c entity.Competition
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&c.Id,
		&c.Name,
		&c.StartTime,
		&c.EndTime,
		&c.FreezeTime,
		&c.IsPaused,
		&c.IsPublic,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entityError.ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("CompetitionRepo - Get: %w", err)
	}

	return &c, nil
}

func (r *CompetitionRepo) Update(ctx context.Context, c *entity.Competition) error {
	query := squirrel.Update("competition").
		Set("name", c.Name).
		Set("start_time", c.StartTime).
		Set("end_time", c.EndTime).
		Set("freeze_time", c.FreezeTime).
		Set("is_paused", c.IsPaused).
		Set("is_public", c.IsPublic).
		Where(squirrel.Eq{"id": 1})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update - BuildQuery: %w", err)
	}

	result, err := r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update - RowsAffected: %w", err)
	}
	if rowsAffected == 0 {
		return entityError.ErrCompetitionNotFound
	}

	return nil
}
