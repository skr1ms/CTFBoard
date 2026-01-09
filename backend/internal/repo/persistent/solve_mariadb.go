package persistent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type SolveRepo struct {
	db *sql.DB
}

func NewSolveRepo(db *sql.DB) *SolveRepo {
	return &SolveRepo{db: db}
}

func (r *SolveRepo) Create(ctx context.Context, s *entity.Solve) error {
	query := squirrel.Insert("solves").
		Columns("id", "user_id", "team_id", "challenge_id", "solved_at").
		Values(uuid.New().String(), uuid.MustParse(s.UserId), uuid.MustParse(s.TeamId), uuid.MustParse(s.ChallengeId), time.Now())

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("SolveRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("SolveRepo - Create - ExecQuery: %w", err)
	}

	return nil
}

func (r *SolveRepo) GetByID(ctx context.Context, id string) (*entity.Solve, error) {
	query := squirrel.Select("id", "user_id", "team_id", "challenge_id", "solved_at").
		From("solves").
		Where(squirrel.Eq{"id": uuid.MustParse(id)})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByID - BuildQuery: %w", err)
	}

	var solve entity.Solve
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&solve.Id,
		&solve.UserId,
		&solve.TeamId,
		&solve.ChallengeId,
		&solve.SolvedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByID - Scan: %w", err)
	}

	return &solve, nil
}

func (r *SolveRepo) GetByTeamAndChallenge(ctx context.Context, teamId, challengeId string) (*entity.Solve, error) {
	query := squirrel.Select("id", "user_id", "team_id", "challenge_id", "solved_at").
		From("solves").
		Where(squirrel.Eq{"team_id": uuid.MustParse(teamId), "challenge_id": uuid.MustParse(challengeId)})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallenge - BuildQuery: %w", err)
	}

	var solve entity.Solve
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&solve.Id,
		&solve.UserId,
		&solve.TeamId,
		&solve.ChallengeId,
		&solve.SolvedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallenge - Scan: %w", err)
	}

	return &solve, nil
}

func (r *SolveRepo) GetByUserId(ctx context.Context, userId string) ([]*entity.Solve, error) {
	query := squirrel.Select("id", "user_id", "team_id", "challenge_id", "solved_at").
		From("solves").
		Where(squirrel.Eq{"user_id": uuid.MustParse(userId)}).
		OrderBy("solved_at DESC")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserId - BuildQuery: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserId - Query: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	solves := make([]*entity.Solve, 0)
	for rows.Next() {
		var solve entity.Solve
		if err := rows.Scan(
			&solve.Id,
			&solve.UserId,
			&solve.TeamId,
			&solve.ChallengeId,
			&solve.SolvedAt,
		); err != nil {
			return nil, fmt.Errorf("SolveRepo - GetByUserId - Scan: %w", err)
		}
		solves = append(solves, &solve)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserId - Rows: %w", err)
	}

	return solves, nil
}

func (r *SolveRepo) GetScoreboard(ctx context.Context) ([]*repo.ScoreboardEntry, error) {
	query := squirrel.Select("t.id", "t.name", "COALESCE(SUM(c.points), 0) as points", "MAX(s.solved_at) as solved_at").
		From("teams t").
		LeftJoin("solves s ON s.team_id = t.id").
		LeftJoin("challenges c ON c.id = s.challenge_id").
		GroupBy("t.id", "t.name").
		OrderBy("points DESC", "solved_at ASC")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboard - BuildQuery: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboard - Query: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	entries := make([]*repo.ScoreboardEntry, 0)
	for rows.Next() {
		var entry repo.ScoreboardEntry
		var solvedAt sql.NullTime
		if err := rows.Scan(
			&entry.TeamId,
			&entry.TeamName,
			&entry.Points,
			&solvedAt,
		); err != nil {
			return nil, fmt.Errorf("SolveRepo - GetScoreboard - Scan: %w", err)
		}
		if solvedAt.Valid {
			entry.SolvedAt = solvedAt.Time
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboard - Rows: %w", err)
	}

	return entries, nil
}
