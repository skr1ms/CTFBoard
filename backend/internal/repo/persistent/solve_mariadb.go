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
	userID, err := uuid.Parse(s.UserId)
	if err != nil {
		return fmt.Errorf("SolveRepo - Create - Parse UserID: %w", err)
	}
	teamID, err := uuid.Parse(s.TeamId)
	if err != nil {
		return fmt.Errorf("SolveRepo - Create - Parse TeamID: %w", err)
	}
	challengeID, err := uuid.Parse(s.ChallengeId)
	if err != nil {
		return fmt.Errorf("SolveRepo - Create - Parse ChallengeID: %w", err)
	}

	query := squirrel.Insert("solves").
		Columns("id", "user_id", "team_id", "challenge_id", "solved_at").
		Values(uuid.New().String(), userID, teamID, challengeID, time.Now())

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
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByID - ParseID: %w", err)
	}

	query := squirrel.Select("id", "user_id", "team_id", "challenge_id", "solved_at").
		From("solves").
		Where(squirrel.Eq{"id": uuidID})

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
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallenge - Parse TeamID: %w", err)
	}
	challengeUUID, err := uuid.Parse(challengeId)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallenge - Parse ChallengeID: %w", err)
	}

	query := squirrel.Select("id", "user_id", "team_id", "challenge_id", "solved_at").
		From("solves").
		Where(squirrel.Eq{"team_id": teamUUID, "challenge_id": challengeUUID})

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
	userUUID, err := uuid.Parse(userId)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserId - ParseID: %w", err)
	}

	query := squirrel.Select("id", "user_id", "team_id", "challenge_id", "solved_at").
		From("solves").
		Where(squirrel.Eq{"user_id": userUUID}).
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

func (r *SolveRepo) GetScoreboardFrozen(ctx context.Context, freezeTime time.Time) ([]*repo.ScoreboardEntry, error) {
	query := squirrel.Select("t.id", "t.name", "COALESCE(SUM(c.points), 0) as points", "MAX(s.solved_at) as solved_at").
		From("teams t").
		LeftJoin("solves s ON s.team_id = t.id AND s.solved_at <= ?", freezeTime).
		LeftJoin("challenges c ON c.id = s.challenge_id").
		GroupBy("t.id", "t.name").
		OrderBy("points DESC", "solved_at ASC")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboardFrozen - BuildQuery: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboardFrozen - Query: %w", err)
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
			return nil, fmt.Errorf("SolveRepo - GetScoreboardFrozen - Scan: %w", err)
		}
		if solvedAt.Valid {
			entry.SolvedAt = solvedAt.Time
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboardFrozen - Rows: %w", err)
	}

	return entries, nil
}

func (r *SolveRepo) GetFirstBlood(ctx context.Context, challengeId string) (*repo.FirstBloodEntry, error) {
	challengeUUID, err := uuid.Parse(challengeId)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetFirstBlood - Parse ChallengeID: %w", err)
	}

	query := squirrel.Select("s.user_id", "u.username", "s.team_id", "t.name", "s.solved_at").
		From("solves s").
		Join("users u ON u.id = s.user_id").
		Join("teams t ON t.id = s.team_id").
		Where(squirrel.Eq{"s.challenge_id": challengeUUID}).
		OrderBy("s.solved_at ASC").
		Limit(1)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetFirstBlood - BuildQuery: %w", err)
	}

	var entry repo.FirstBloodEntry
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&entry.UserId,
		&entry.Username,
		&entry.TeamId,
		&entry.TeamName,
		&entry.SolvedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetFirstBlood - Scan: %w", err)
	}

	return &entry, nil
}
