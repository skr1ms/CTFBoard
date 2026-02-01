package persistent

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

var challengeColumns = []string{
	"id", "title", "description", "category", "points", "initial_value", "min_value", "decay",
	"solve_count", "flag_hash", "is_hidden", "is_regex", "is_case_insensitive", "flag_regex", "flag_format_regex",
}

func scanChallenge(row rowScanner) (*entity.Challenge, error) {
	var c entity.Challenge
	err := row.Scan(
		&c.ID, &c.Title, &c.Description, &c.Category, &c.Points, &c.InitialValue, &c.MinValue, &c.Decay,
		&c.SolveCount, &c.FlagHash, &c.IsHidden, &c.IsRegex, &c.IsCaseInsensitive, &c.FlagRegex, &c.FlagFormatRegex,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanChallengeWithSolved(row rowScanner) (*entity.Challenge, int, error) {
	var c entity.Challenge
	var solved int
	err := row.Scan(
		&c.ID, &c.Title, &c.Description, &c.Category, &c.Points, &c.InitialValue, &c.MinValue, &c.Decay,
		&c.SolveCount, &c.FlagHash, &c.IsHidden, &c.IsRegex, &c.IsCaseInsensitive, &c.FlagRegex, &c.FlagFormatRegex,
		&solved,
	)
	if err != nil {
		return nil, 0, err
	}
	return &c, solved, nil
}

type ChallengeRepo struct {
	pool *pgxpool.Pool
}

func NewChallengeRepo(pool *pgxpool.Pool) *ChallengeRepo {
	return &ChallengeRepo{pool: pool}
}

func (r *ChallengeRepo) Create(ctx context.Context, c *entity.Challenge) error {
	c.ID = uuid.New()

	query := squirrel.Insert("challenges").
		Columns(challengeColumns...).
		Values(c.ID, c.Title, c.Description, c.Category, c.Points, c.InitialValue, c.MinValue, c.Decay, c.SolveCount, c.FlagHash, c.IsHidden, c.IsRegex, c.IsCaseInsensitive, c.FlagRegex, c.FlagFormatRegex).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Create - ExecQuery: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Challenge, error) {
	query := squirrel.Select(challengeColumns...).
		From("challenges").
		Where(squirrel.Eq{"id": ID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetByID - BuildQuery: %w", err)
	}

	challenge, err := scanChallenge(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ChallengeRepo - GetByID - Scan: %w", err)
	}
	return challenge, nil
}

func challengeColumnsC() []string {
	out := make([]string, len(challengeColumns))
	for i, col := range challengeColumns {
		out[i] = "c." + col
	}
	return out
}

func (r *ChallengeRepo) GetAll(ctx context.Context, teamID *uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	var query squirrel.SelectBuilder
	colsC := challengeColumnsC()

	if teamID != nil {
		query = squirrel.Select(append(colsC, "CASE WHEN s.id IS NOT NULL THEN 1 ELSE 0 END as solved")...).
			From("challenges c").
			LeftJoin("solves s ON c.id = s.challenge_id AND s.team_id = ?", *teamID).
			Where(squirrel.Eq{"c.is_hidden": false})
	} else {
		query = squirrel.Select(append(colsC, "0 as solved")...).
			From("challenges c").
			Where(squirrel.Eq{"c.is_hidden": false})
	}

	sqlQuery, args, err := query.PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAll - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAll - Query: %w", err)
	}
	defer rows.Close()

	result := make([]*repo.ChallengeWithSolved, 0)
	for rows.Next() {
		challenge, solved, err := scanChallengeWithSolved(rows)
		if err != nil {
			return nil, fmt.Errorf("ChallengeRepo - GetAll - Scan: %w", err)
		}
		result = append(result, &repo.ChallengeWithSolved{
			Challenge: challenge,
			Solved:    solved == 1,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAll - Rows: %w", err)
	}

	return result, nil
}

func (r *ChallengeRepo) Update(ctx context.Context, c *entity.Challenge) error {
	query := squirrel.Update("challenges").
		Set("title", c.Title).
		Set("description", c.Description).
		Set("category", c.Category).
		Set("points", c.Points).
		Set("initial_value", c.InitialValue).
		Set("min_value", c.MinValue).
		Set("decay", c.Decay).
		Set("flag_hash", c.FlagHash).
		Set("is_hidden", c.IsHidden).
		Set("is_regex", c.IsRegex).
		Set("is_case_insensitive", c.IsCaseInsensitive).
		Set("flag_regex", c.FlagRegex).
		Set("flag_format_regex", c.FlagFormatRegex).
		Where(squirrel.Eq{"id": c.ID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Update - ExecQuery: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	query := squirrel.Delete("challenges").
		Where(squirrel.Eq{"id": ID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Delete - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - Delete - ExecQuery: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) IncrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error) {
	query := squirrel.Update("challenges").
		Set("solve_count", squirrel.Expr("solve_count + 1")).
		Where(squirrel.Eq{"id": ID}).
		Suffix("RETURNING solve_count").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCount - BuildQuery: %w", err)
	}

	var solveCount int
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(&solveCount)
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCount - Query: %w", err)
	}

	return solveCount, nil
}

func (r *ChallengeRepo) UpdatePoints(ctx context.Context, ID uuid.UUID, points int) error {
	query := squirrel.Update("challenges").
		Set("points", points).
		Where(squirrel.Eq{"id": ID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePoints - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePoints - Exec: %w", err)
	}

	return nil
}
