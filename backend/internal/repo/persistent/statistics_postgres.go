package persistent

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

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
	query := squirrel.Select("id", "title", "category", "points", "solve_count").
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
		s := &entity.ChallengeStats{}
		if err := rows.Scan(&s.ID, &s.Title, &s.Category, &s.Points, &s.SolveCount); err != nil {
			return nil, fmt.Errorf("StatisticsRepository - GetChallengeStats - Scan: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, nil
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
