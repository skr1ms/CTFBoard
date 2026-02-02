package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type StatisticsRepository struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewStatisticsRepository(db *pgxpool.Pool) *StatisticsRepository {
	return &StatisticsRepository{db: db, q: sqlc.New(db)}
}

func (r *StatisticsRepository) GetGeneralStats(ctx context.Context) (*entity.GeneralStats, error) {
	users, err := r.q.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetGeneralStats - CountUsers: %w", err)
	}
	teams, err := r.q.CountTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetGeneralStats - CountTeams: %w", err)
	}
	challenges, err := r.q.CountChallenges(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetGeneralStats - CountChallenges: %w", err)
	}
	solves, err := r.q.CountSolves(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetGeneralStats - CountSolves: %w", err)
	}
	return &entity.GeneralStats{
		UserCount:      int(users),
		TeamCount:      int(teams),
		ChallengeCount: int(challenges),
		SolveCount:     int(solves),
	}, nil
}

func (r *StatisticsRepository) GetChallengeStats(ctx context.Context) ([]*entity.ChallengeStats, error) {
	rows, err := r.q.GetChallengeStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetChallengeStats: %w", err)
	}
	out := make([]*entity.ChallengeStats, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.ChallengeStats{
			ID:         row.ID,
			Title:      row.Title,
			Category:   ptrStrToStr(row.Category),
			Points:     int32PtrToInt(row.Points),
			SolveCount: int(row.SolveCount),
		})
	}
	return out, nil
}

func (r *StatisticsRepository) GetChallengeDetailStats(ctx context.Context, challengeID uuid.UUID) (*entity.ChallengeDetailStats, error) {
	chRow, err := r.q.GetChallengeDetailChallenge(ctx, challengeID)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("StatisticsRepository - GetChallengeDetailStats - Challenge: %w", err)
	}
	percentageSolved := 0.0
	if chRow.TotalTeams > 0 {
		percentageSolved = float64(chRow.SolveCount) / float64(chRow.TotalTeams) * 100
	}
	solveRows, err := r.q.GetChallengeDetailSolves(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetChallengeDetailStats - Solves: %w", err)
	}
	solves := make([]entity.ChallengeSolveEntry, 0, len(solveRows))
	for _, row := range solveRows {
		solves = append(solves, entity.ChallengeSolveEntry{
			TeamID:   row.TeamID,
			TeamName: row.TeamName,
			SolvedAt: ptrTimeToTime(row.SolvedAt),
		})
	}
	var firstBlood *entity.ChallengeSolveEntry
	if len(solves) > 0 {
		firstBlood = &solves[0]
	}
	return &entity.ChallengeDetailStats{
		ID:               chRow.ID,
		Title:            chRow.Title,
		Category:         ptrStrToStr(chRow.Category),
		Points:           int32PtrToInt(chRow.Points),
		SolveCount:       int(chRow.SolveCount),
		TotalTeams:       int(chRow.TotalTeams),
		PercentageSolved: percentageSolved,
		FirstBlood:       firstBlood,
		Solves:           solves,
	}, nil
}

func (r *StatisticsRepository) GetScoreboardHistory(ctx context.Context, limit int) ([]*entity.ScoreboardHistoryEntry, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetScoreboardHistory limit: %w", err)
	}
	rows, err := r.q.GetScoreboardHistory(ctx, limit32)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepository - GetScoreboardHistory: %w", err)
	}
	out := make([]*entity.ScoreboardHistoryEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.ScoreboardHistoryEntry{
			TeamID:    row.TeamID,
			TeamName:  row.TeamName,
			Points:    int(row.Points),
			Timestamp: ptrTimeToTime(row.Timestamp),
		})
	}
	return out, nil
}

var _ repo.StatisticsRepository = (*StatisticsRepository)(nil)
