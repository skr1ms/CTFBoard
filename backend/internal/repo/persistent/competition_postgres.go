package persistent

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
)

type CompetitionRepo struct {
	pool *pgxpool.Pool
}

func NewCompetitionRepo(pool *pgxpool.Pool) *CompetitionRepo {
	return &CompetitionRepo{pool: pool}
}

func (r *CompetitionRepo) Get(ctx context.Context) (*entity.Competition, error) {
	query := squirrel.Select("id", "name", "start_time", "end_time", "freeze_time", "is_paused", "is_public", "flag_regex", "mode", "allow_team_switch", "min_team_size", "max_team_size", "created_at", "updated_at").
		From("competition").
		Where(squirrel.Eq{"id": 1}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("CompetitionRepo - Get - BuildQuery: %w", err)
	}

	var c entity.Competition
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(
		&c.Id,
		&c.Name,
		&c.StartTime,
		&c.EndTime,
		&c.FreezeTime,
		&c.IsPaused,
		&c.IsPublic,
		&c.FlagRegex,
		&c.Mode,
		&c.AllowTeamSwitch,
		&c.MinTeamSize,
		&c.MaxTeamSize,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
		Set("flag_regex", c.FlagRegex).
		Set("mode", c.Mode).
		Set("allow_team_switch", c.AllowTeamSwitch).
		Set("min_team_size", c.MinTeamSize).
		Set("max_team_size", c.MaxTeamSize).
		Where(squirrel.Eq{"id": 1}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update - BuildQuery: %w", err)
	}

	result, err := r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("CompetitionRepo - Update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return entityError.ErrCompetitionNotFound
	}

	return nil
}
