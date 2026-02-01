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
	"github.com/skr1ms/CTFBoard/internal/repo"
)

var statisticsChallengeStatsColumns = []string{"id", "title", "category", "points", "solve_count"}

var statisticsChallengeDetailColumns = []string{
	"c.id", "c.title", "c.category", "c.points", "c.solve_count",
	"(SELECT COUNT(*)::int FROM teams WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false) AS total_teams",
}

var statisticsSolveEntryColumns = []string{"s.team_id", "t.name", "s.solved_at"}

func scanChallengeStats(row rowScanner) (*entity.ChallengeStats, error) {
	var s entity.ChallengeStats
	err := row.Scan(&s.ID, &s.Title, &s.Category, &s.Points, &s.SolveCount)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

type StatisticsRepository struct {
	pool *pgxpool.Pool
}

func NewStatisticsRepository(pool *pgxpool.Pool) *StatisticsRepository {
	return &StatisticsRepository{pool: pool}
}

func (r *StatisticsRepository) GetGeneralStats(ctx context.Context) (*entity.GeneralStats, error) {
	stats := &entity.GeneralStats{}
	batch := &pgx.Batch{}

	// Queue queries in order
	batch.Queue("SELECT COUNT(*) FROM users")
	batch.Queue("SELECT COUNT(*) FROM teams WHERE deleted_at IS NULL")
	batch.Queue("SELECT COUNT(*) FROM challenges")
	batch.Queue("SELECT COUNT(*) FROM solves")

	br := r.pool.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()

	var err error

	// Read results in order
	if err = br.QueryRow().Scan(&stats.UserCount); err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetGeneralStats - ScanUsers: %w", err)
	}
	if err = br.QueryRow().Scan(&stats.TeamCount); err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetGeneralStats - ScanTeams: %w", err)
	}
	if err = br.QueryRow().Scan(&stats.ChallengeCount); err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetGeneralStats - ScanChallenges: %w", err)
	}
	if err = br.QueryRow().Scan(&stats.SolveCount); err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetGeneralStats - ScanSolves: %w", err)
	}

	return stats, nil
}

func (r *StatisticsRepository) GetChallengeStats(ctx context.Context) ([]*entity.ChallengeStats, error) {
	query := squirrel.Select(statisticsChallengeStatsColumns...).
		From("challenges").
		OrderBy("solve_count DESC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetChallengeStats - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetChallengeStats - Query: %w", err)
	}
	defer rows.Close()

	var stats []*entity.ChallengeStats
	for rows.Next() {
		s, err := scanChallengeStats(rows)
		if err != nil {
			return nil, fmt.Errorf("StatisticsRepository - GetChallengeStats - Scan: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *StatisticsRepository) GetChallengeDetailStats(ctx context.Context, challengeID uuid.UUID) (*entity.ChallengeDetailStats, error) {
	challengeQuery := squirrel.Select(statisticsChallengeDetailColumns...).
		From("challenges c").
		Where(squirrel.Eq{"c.id": challengeID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := challengeQuery.ToSql()
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetChallengeDetailStats - BuildChallengeQuery: %w", err)
	}

	var id uuid.UUID
	var title, category string
	var points, solveCount, totalTeams int
	err = r.pool.QueryRow(ctx, sqlQuery, args...).Scan(&id, &title, &category, &points, &solveCount, &totalTeams)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("StatisticsRepository - GetChallengeDetailStats - Challenge: %w", err)
	}

	percentageSolved := 0.0
	if totalTeams > 0 {
		percentageSolved = float64(solveCount) / float64(totalTeams) * 100
	}

	solvesQuery := squirrel.Select(statisticsSolveEntryColumns...).
		From("solves s").
		Join("teams t ON t.id = s.team_id").
		Where(squirrel.Eq{"s.challenge_id": challengeID}).
		Where(squirrel.Eq{"t.deleted_at": nil}).
		Where(squirrel.Eq{"t.is_banned": false}).
		Where(squirrel.Eq{"t.is_hidden": false}).
		OrderBy("s.solved_at ASC").
		PlaceholderFormat(squirrel.Dollar)

	solvesSQL, solvesArgs, err := solvesQuery.ToSql()
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetChallengeDetailStats - BuildSolvesQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, solvesSQL, solvesArgs...)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetChallengeDetailStats - Solves: %w", err)
	}
	defer rows.Close()

	var solves []entity.ChallengeSolveEntry
	for rows.Next() {
		var e entity.ChallengeSolveEntry
		if err := rows.Scan(&e.TeamID, &e.TeamName, &e.SolvedAt); err != nil {
			return nil, fmt.Errorf("StatisticsRepository - GetChallengeDetailStats - Scan: %w", err)
		}
		solves = append(solves, e)
	}

	var firstBlood *entity.ChallengeSolveEntry
	if len(solves) > 0 {
		firstBlood = &solves[0]
	}

	return &entity.ChallengeDetailStats{
		ID:               id,
		Title:            title,
		Category:         category,
		Points:           points,
		SolveCount:       solveCount,
		TotalTeams:       totalTeams,
		PercentageSolved: percentageSolved,
		FirstBlood:       firstBlood,
		Solves:           solves,
	}, nil
}

func (r *StatisticsRepository) GetScoreboardHistory(ctx context.Context, limit int) ([]*entity.ScoreboardHistoryEntry, error) {
	query := `
		WITH top_teams AS (
			SELECT t.id, t.name
			FROM teams t
			LEFT JOIN solves s ON s.team_id = t.id
			LEFT JOIN challenges c ON s.challenge_id = c.id
			LEFT JOIN awards a ON a.team_id = t.id
			WHERE t.deleted_at IS NULL
			GROUP BY t.id
			ORDER BY COALESCE(SUM(c.points), 0) + COALESCE(SUM(a.value), 0) DESC
			LIMIT $1
		),
		events AS (
			SELECT 
				s.team_id,
				s.solved_at as event_time,
				c.points as delta
			FROM solves s
			JOIN challenges c ON s.challenge_id = c.id
			WHERE s.team_id IN (SELECT id FROM top_teams)
			
			UNION ALL
			
			SELECT 
				a.team_id,
				a.created_at as event_time,
				a.value as delta
			FROM awards a
			WHERE a.team_id IN (SELECT id FROM top_teams)
		)
		SELECT 
			e.team_id,
			tt.name,
			SUM(e.delta) OVER (PARTITION BY e.team_id ORDER BY e.event_time) as running_total,
			e.event_time
		FROM events e
		JOIN top_teams tt ON e.team_id = tt.id
		ORDER BY e.team_id, e.event_time
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetScoreboardHistory - Query: %w", err)
	}
	defer rows.Close()

	var history []*entity.ScoreboardHistoryEntry
	for rows.Next() {
		var h entity.ScoreboardHistoryEntry
		if err := rows.Scan(&h.TeamID, &h.TeamName, &h.Points, &h.Timestamp); err != nil {
			return nil, fmt.Errorf("StatisticsRepository - GetScoreboardHistory - Scan: %w", err)
		}
		history = append(history, &h)
	}

	return history, nil
}

var _ repo.StatisticsRepository = (*StatisticsRepository)(nil)
