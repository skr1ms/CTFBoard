package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

var solveColumns = []string{"id", "user_id", "team_id", "challenge_id", "solved_at"}

func scanSolve(row rowScanner) (*entity.Solve, error) {
	var s entity.Solve
	err := row.Scan(&s.ID, &s.UserID, &s.TeamID, &s.ChallengeID, &s.SolvedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

type SolveRepo struct {
	pool *pgxpool.Pool
}

func NewSolveRepo(pool *pgxpool.Pool) *SolveRepo {
	return &SolveRepo{pool: pool}
}

func (r *SolveRepo) Create(ctx context.Context, s *entity.Solve) error {
	s.ID = uuid.New()
	s.SolvedAt = time.Now()

	query := squirrel.Insert("solves").
		Columns(solveColumns...).
		Values(s.ID, s.UserID, s.TeamID, s.ChallengeID, s.SolvedAt).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("SolveRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("SolveRepo - Create - ExecQuery: %w", err)
	}

	return nil
}

func (r *SolveRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Solve, error) {
	query := squirrel.Select(solveColumns...).
		From("solves").
		Where(squirrel.Eq{"id": ID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByID - BuildQuery: %w", err)
	}

	solve, err := scanSolve(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByID - Scan: %w", err)
	}
	return solve, nil
}

func (r *SolveRepo) GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*entity.Solve, error) {
	query := squirrel.Select(solveColumns...).
		From("solves").
		Where(squirrel.Eq{"team_id": teamID, "challenge_id": challengeID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallenge - BuildQuery: %w", err)
	}

	solve, err := scanSolve(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallenge - Scan: %w", err)
	}
	return solve, nil
}

func (r *SolveRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Solve, error) {
	query := squirrel.Select(solveColumns...).
		From("solves").
		Where(squirrel.Eq{"user_id": userID}).
		OrderBy("solved_at DESC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserID - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserID - Query: %w", err)
	}
	defer rows.Close()

	solves := make([]*entity.Solve, 0)
	for rows.Next() {
		solve, err := scanSolve(rows)
		if err != nil {
			return nil, fmt.Errorf("SolveRepo - GetByUserID - Scan: %w", err)
		}
		solves = append(solves, solve)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserID - Rows: %w", err)
	}

	return solves, nil
}

func (r *SolveRepo) GetScoreboard(ctx context.Context) ([]*repo.ScoreboardEntry, error) {
	solveSQL, solveArgs, err := r.buildSolvePointsSubquery()
	if err != nil {
		return nil, err
	}

	awardSQL, awardArgs, err := r.buildAwardPointsSubquery()
	if err != nil {
		return nil, err
	}

	query := squirrel.Select(
		"t.id",
		"t.name",
		"COALESCE(solve_points.points, 0) + COALESCE(award_points.total, 0) as points",
		"solve_points.last_solved",
	).
		From("teams t").
		LeftJoin(fmt.Sprintf("(%s) solve_points ON solve_points.team_id = t.id", solveSQL), solveArgs...).
		LeftJoin(fmt.Sprintf("(%s) award_points ON award_points.team_id = t.id", awardSQL), awardArgs...).
		Where(squirrel.Eq{"t.is_banned": false}).
		Where(squirrel.Eq{"t.is_hidden": false}).
		Where(squirrel.Eq{"t.deleted_at": nil}).
		OrderBy("points DESC", "COALESCE(solve_points.last_solved, '9999-12-31') ASC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboard - BuildQuery: %w", err)
	}

	return r.executeScoreboardQuery(ctx, sqlQuery, args)
}

func (r *SolveRepo) buildSolvePointsSubquery() (string, []any, error) {
	subquery := squirrel.Select("s.team_id", "SUM(c.points) as points", "MAX(s.solved_at) as last_solved").
		From("solves s").
		Join("challenges c ON c.id = s.challenge_id").
		GroupBy("s.team_id").
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := subquery.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("SolveRepo - GetScoreboard - BuildSolveSubquery: %w", err)
	}
	return sql, args, nil
}

func (r *SolveRepo) buildAwardPointsSubquery() (string, []any, error) {
	subquery := squirrel.Select("team_id", "SUM(value) as total").
		From("awards").
		GroupBy("team_id").
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := subquery.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("SolveRepo - GetScoreboard - BuildAwardSubquery: %w", err)
	}
	return sql, args, nil
}

func (r *SolveRepo) executeScoreboardQuery(ctx context.Context, sqlQuery string, args []any) ([]*repo.ScoreboardEntry, error) {
	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboard - Query: %w", err)
	}
	defer rows.Close()

	entries := make([]*repo.ScoreboardEntry, 0)
	for rows.Next() {
		var entry repo.ScoreboardEntry
		var solvedAt *time.Time
		if err := rows.Scan(
			&entry.TeamID,
			&entry.TeamName,
			&entry.Points,
			&solvedAt,
		); err != nil {
			return nil, fmt.Errorf("SolveRepo - GetScoreboard - Scan: %w", err)
		}
		if solvedAt != nil {
			entry.SolvedAt = *solvedAt
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboard - Rows: %w", err)
	}

	return entries, nil
}

func (r *SolveRepo) GetScoreboardFrozen(ctx context.Context, freezeTime time.Time) ([]*repo.ScoreboardEntry, error) {
	sqlQuery := `
		SELECT 
			t.id,
			t.name,
			COALESCE(solve_points.points, 0) + COALESCE(award_points.total, 0) as points,
			solve_points.last_solved
		FROM teams t
		LEFT JOIN (
			SELECT s.team_id, SUM(c.points) as points, MAX(s.solved_at) as last_solved
			FROM solves s
			JOIN challenges c ON c.id = s.challenge_id
			WHERE s.solved_at <= $1
			GROUP BY s.team_id
		) solve_points ON solve_points.team_id = t.id
		LEFT JOIN (
			SELECT team_id, SUM(value) as total
			FROM awards
			WHERE created_at <= $2
			GROUP BY team_id
		) award_points ON award_points.team_id = t.id
		WHERE t.is_banned = false AND t.is_hidden = false AND t.deleted_at IS NULL
		ORDER BY points DESC, COALESCE(solve_points.last_solved, '9999-12-31') ASC
	`

	rows, err := r.pool.Query(ctx, sqlQuery, freezeTime, freezeTime)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboardFrozen - Query: %w", err)
	}
	defer rows.Close()

	entries := make([]*repo.ScoreboardEntry, 0)
	for rows.Next() {
		var entry repo.ScoreboardEntry
		var solvedAt *time.Time
		if err := rows.Scan(
			&entry.TeamID,
			&entry.TeamName,
			&entry.Points,
			&solvedAt,
		); err != nil {
			return nil, fmt.Errorf("SolveRepo - GetScoreboardFrozen - Scan: %w", err)
		}
		if solvedAt != nil {
			entry.SolvedAt = *solvedAt
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboardFrozen - Rows: %w", err)
	}

	return entries, nil
}

func (r *SolveRepo) GetTeamScore(ctx context.Context, teamID uuid.UUID) (int, error) {
	solveQuery := squirrel.Select("COALESCE(SUM(c.points), 0)").
		From("solves s").
		Join("challenges c ON c.id = s.challenge_id").
		Where(squirrel.Eq{"s.team_id": teamID}).
		PlaceholderFormat(squirrel.Dollar)

	solveSQL, solveArgs, err := solveQuery.ToSql()
	if err != nil {
		return 0, fmt.Errorf("SolveRepo - GetTeamScore - BuildSolveQuery: %w", err)
	}

	var solvePoints int
	err = r.pool.QueryRow(ctx, solveSQL, solveArgs...).Scan(&solvePoints)
	if err != nil {
		return 0, fmt.Errorf("SolveRepo - GetTeamScore - ScanSolves: %w", err)
	}

	awardQuery := squirrel.Select("COALESCE(SUM(value), 0)").
		From("awards").
		Where(squirrel.Eq{"team_id": teamID}).
		PlaceholderFormat(squirrel.Dollar)

	awardSQL, awardArgs, err := awardQuery.ToSql()
	if err != nil {
		return 0, fmt.Errorf("SolveRepo - GetTeamScore - BuildAwardQuery: %w", err)
	}

	var awardPoints int
	err = r.pool.QueryRow(ctx, awardSQL, awardArgs...).Scan(&awardPoints)
	if err != nil {
		return 0, fmt.Errorf("SolveRepo - GetTeamScore - ScanAwards: %w", err)
	}

	return solvePoints + awardPoints, nil
}

func (r *SolveRepo) GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*repo.FirstBloodEntry, error) {
	query := squirrel.Select("s.user_id", "u.username", "s.team_id", "t.name", "s.solved_at").
		From("solves s").
		Join("users u ON u.id = s.user_id").
		Join("teams t ON t.id = s.team_id").
		Where(squirrel.Eq{"s.challenge_id": challengeID}).
		OrderBy("s.solved_at ASC").
		Limit(1).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetFirstBlood - BuildQuery: %w", err)
	}

	var entry repo.FirstBloodEntry
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(
		&entry.UserID,
		&entry.Username,
		&entry.TeamID,
		&entry.TeamName,
		&entry.SolvedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetFirstBlood - Scan: %w", err)
	}

	return &entry, nil
}

func (r *SolveRepo) GetAll(ctx context.Context) ([]*entity.Solve, error) {
	query := squirrel.Select(solveColumns...).
		From("solves").
		OrderBy("solved_at ASC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetAll - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetAll - Query: %w", err)
	}
	defer rows.Close()

	var solves []*entity.Solve
	for rows.Next() {
		solve, err := scanSolve(rows)
		if err != nil {
			return nil, fmt.Errorf("SolveRepo - GetAll - Scan: %w", err)
		}
		solves = append(solves, solve)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SolveRepo - GetAll - Rows: %w", err)
	}

	return solves, nil
}
