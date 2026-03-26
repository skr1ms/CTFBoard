package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const dateFormatISO = time.DateOnly

type StatisticsRepo struct {
	BaseRepo
}

var _ repo.StatisticsRepository = (*StatisticsRepo)(nil)

func NewStatisticsRepo(pool *pgxpool.Pool) *StatisticsRepo {
	return &StatisticsRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *StatisticsRepo) GetGeneralStats(ctx context.Context) (*domain.GeneralStats, error) {
	var users, teams, challenges, solves int32

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		v, err := r.Q(gCtx).CountUsers(gCtx)
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStats - CountUsers: %w", err)
		}

		users = v

		return nil
	})
	g.Go(func() error {
		v, err := r.Q(gCtx).CountTeams(gCtx)
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStats - CountTeams: %w", err)
		}

		teams = v

		return nil
	})
	g.Go(func() error {
		v, err := r.Q(gCtx).CountChallenges(gCtx)
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStats - CountChallenges: %w", err)
		}

		challenges = v

		return nil
	})
	g.Go(func() error {
		v, err := r.Q(gCtx).CountSolves(gCtx)
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStats - CountSolves: %w", err)
		}

		solves = v

		return nil
	})

	err := g.Wait()
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetGeneralStats - Wait: %w", err)
	}

	return &domain.GeneralStats{
		UserCount:      int(users),
		TeamCount:      int(teams),
		ChallengeCount: int(challenges),
		SolveCount:     int(solves),
	}, nil
}

func (r *StatisticsRepo) GetGeneralStatsFrozen(ctx context.Context, freezeTime time.Time) (*domain.GeneralStats, error) {
	var users, teams, challenges, solves int32

	ft := &freezeTime

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		v, err := r.Q(gCtx).CountUsersFrozen(gCtx, pgutil.TimeToTimestamptz(ft))
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStatsFrozen - CountUsersFrozen: %w", err)
		}

		users = v

		return nil
	})
	g.Go(func() error {
		v, err := r.Q(gCtx).CountTeamsFrozen(gCtx, pgutil.TimeToTimestamptz(ft))
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStatsFrozen - CountTeamsFrozen: %w", err)
		}

		teams = v

		return nil
	})
	g.Go(func() error {
		v, err := r.Q(gCtx).CountChallenges(gCtx)
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStatsFrozen - CountChallenges: %w", err)
		}

		challenges = v

		return nil
	})
	g.Go(func() error {
		v, err := r.Q(gCtx).CountSolvesFrozen(gCtx, pgutil.TimeToTimestamptz(&freezeTime))
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStatsFrozen - CountSolvesFrozen: %w", err)
		}

		solves = v

		return nil
	})

	err := g.Wait()
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetGeneralStatsFrozen - Wait: %w", err)
	}

	return &domain.GeneralStats{
		UserCount:      int(users),
		TeamCount:      int(teams),
		ChallengeCount: int(challenges),
		SolveCount:     int(solves),
	}, nil
}

func (r *StatisticsRepo) GetChallengeStats(ctx context.Context) ([]*domain.ChallengeStats, error) {
	rows, err := r.Q(ctx).GetChallengeStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeStats: %w", err)
	}

	out := make([]*domain.ChallengeStats, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ChallengeStats{
			ID:         row.ID,
			Title:      row.Title,
			Category:   row.Category,
			Points:     int(lo.FromPtr(row.Points)),
			SolveCount: int(row.SolveCount),
		})
	}

	return out, nil
}

func (r *StatisticsRepo) GetChallengeStatsFrozen(ctx context.Context, freezeTime time.Time) ([]*domain.ChallengeStats, error) {
	rows, err := r.Q(ctx).GetChallengeStatsFrozen(ctx, pgutil.TimeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeStatsFrozen: %w", err)
	}

	out := make([]*domain.ChallengeStats, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ChallengeStats{
			ID:         row.ID,
			Title:      row.Title,
			Category:   row.Category,
			Points:     int(lo.FromPtr(row.Points)),
			SolveCount: int(row.SolveCount),
		})
	}

	return out, nil
}

func (r *StatisticsRepo) GetChallengeDetailStats(ctx context.Context, challengeID uuid.UUID) (*domain.ChallengeDetailStats, error) {
	chRow, err := r.Q(ctx).GetChallengeDetailChallenge(ctx, challengeID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("StatisticsRepo - GetChallengeDetailStats - Challenge: %w", err)
	}

	percentageSolved := 0.0

	if chRow.TotalTeams > 0 {
		percentageSolved = float64(chRow.SolveCount) / float64(chRow.TotalTeams) * 100
	}

	solveRows, err := r.Q(ctx).GetChallengeDetailSolves(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeDetailStats - Solves: %w", err)
	}

	solves := make([]domain.ChallengeSolveEntry, 0, len(solveRows))
	for _, row := range solveRows {
		solves = append(solves, domain.ChallengeSolveEntry{
			TeamID:   row.TeamID,
			TeamName: row.TeamName,
			SolvedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.SolvedAt)),
		})
	}

	var firstBlood *domain.ChallengeSolveEntry

	if len(solves) > 0 {
		firstBlood = &solves[0]
	}

	return &domain.ChallengeDetailStats{
		ID:               chRow.ID,
		Title:            chRow.Title,
		Category:         chRow.Category,
		Points:           int(lo.FromPtr(chRow.Points)),
		SolveCount:       int(chRow.SolveCount),
		TotalTeams:       int(chRow.TotalTeams),
		PercentageSolved: percentageSolved,
		FirstBlood:       firstBlood,
		Solves:           solves,
	}, nil
}

func (r *StatisticsRepo) GetChallengeDetailStatsFrozen(ctx context.Context, challengeID uuid.UUID, freezeTime time.Time) (*domain.ChallengeDetailStats, error) {
	chRow, err := r.Q(ctx).GetChallengeDetailChallengeFrozen(ctx, sqlc.GetChallengeDetailChallengeFrozenParams{
		ID:       challengeID,
		SolvedAt: pgutil.TimeToTimestamptz(&freezeTime),
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("StatisticsRepo - GetChallengeDetailStatsFrozen - Challenge: %w", err)
	}

	percentageSolved := 0.0

	if chRow.TotalTeams > 0 {
		percentageSolved = float64(chRow.SolveCount) / float64(chRow.TotalTeams) * 100
	}

	solveRows, err := r.Q(ctx).GetChallengeDetailSolvesFrozen(ctx, sqlc.GetChallengeDetailSolvesFrozenParams{
		ChallengeID: challengeID,
		SolvedAt:    pgutil.TimeToTimestamptz(&freezeTime),
	})
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeDetailStatsFrozen - Solves: %w", err)
	}

	solves := make([]domain.ChallengeSolveEntry, 0, len(solveRows))
	for _, row := range solveRows {
		solves = append(solves, domain.ChallengeSolveEntry{
			TeamID:   row.TeamID,
			TeamName: row.TeamName,
			SolvedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.SolvedAt)),
		})
	}

	var firstBlood *domain.ChallengeSolveEntry

	if len(solves) > 0 {
		firstBlood = &solves[0]
	}

	return &domain.ChallengeDetailStats{
		ID:               chRow.ID,
		Title:            chRow.Title,
		Category:         chRow.Category,
		Points:           int(lo.FromPtr(chRow.Points)),
		SolveCount:       int(chRow.SolveCount),
		TotalTeams:       int(chRow.TotalTeams),
		PercentageSolved: percentageSolved,
		FirstBlood:       firstBlood,
		Solves:           solves,
	}, nil
}

func (r *StatisticsRepo) GetScoreboardHistory(ctx context.Context, limit int) ([]*domain.ScoreboardHistoryEntry, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreboardHistory - limit: %w", err)
	}

	rows, err := r.Q(ctx).GetScoreboardHistory(ctx, limit32)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreboardHistory: %w", err)
	}

	return mapScoreboardHistoryRows(rows), nil
}

func (r *StatisticsRepo) GetScoreboardHistoryFrozen(ctx context.Context, freezeTime time.Time, limit int) ([]*domain.ScoreboardHistoryEntry, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreboardHistoryFrozen - limit: %w", err)
	}

	rows, err := r.Q(ctx).GetScoreboardHistoryFrozen(ctx, sqlc.GetScoreboardHistoryFrozenParams{
		Limit:    limit32,
		SolvedAt: pgutil.TimeToTimestamptz(&freezeTime),
	})
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreboardHistoryFrozen: %w", err)
	}

	out := make([]*domain.ScoreboardHistoryEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ScoreboardHistoryEntry{
			TeamID:    row.TeamID,
			TeamName:  row.TeamName,
			Points:    int(row.Points),
			Timestamp: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.Timestamp)),
		})
	}

	return out, nil
}

func mapScoreboardHistoryRows(rows []sqlc.GetScoreboardHistoryRow) []*domain.ScoreboardHistoryEntry {
	out := make([]*domain.ScoreboardHistoryEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ScoreboardHistoryEntry{
			TeamID:    row.TeamID,
			TeamName:  row.TeamName,
			Points:    int(row.Points),
			Timestamp: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.Timestamp)),
		})
	}

	return out
}

func (r *StatisticsRepo) GetChallengeSolvePercentages(ctx context.Context) ([]*domain.ChallengeSolvePercentage, error) {
	rows, err := r.Q(ctx).GetChallengeSolvePercentages(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeSolvePercentages: %w", err)
	}

	return mapChallengeSolvePercentageRows(rows), nil
}

func (r *StatisticsRepo) GetChallengeSolvePercentagesFrozen(ctx context.Context, freezeTime time.Time) ([]*domain.ChallengeSolvePercentage, error) {
	rows, err := r.Q(ctx).GetChallengeSolvePercentagesFrozen(ctx, pgutil.TimeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeSolvePercentagesFrozen: %w", err)
	}

	out := make([]*domain.ChallengeSolvePercentage, 0, len(rows))
	for _, row := range rows {
		pct := 0.0

		if row.Percentage != nil {
			if v, ok := row.Percentage.(float64); ok {
				pct = v
			}
		}

		out = append(out, &domain.ChallengeSolvePercentage{
			ID:         row.ID,
			Title:      row.Title,
			Category:   row.Category,
			SolveCount: int(row.SolveCount),
			TotalTeams: int(row.TotalTeams),
			Percentage: pct,
		})
	}

	return out, nil
}

func mapChallengeSolvePercentageRows(rows []sqlc.GetChallengeSolvePercentagesRow) []*domain.ChallengeSolvePercentage {
	out := make([]*domain.ChallengeSolvePercentage, 0, len(rows))
	for _, row := range rows {
		pct := 0.0

		if row.Percentage != nil {
			if v, ok := row.Percentage.(float64); ok {
				pct = v
			}
		}

		out = append(out, &domain.ChallengeSolvePercentage{
			ID:         row.ID,
			Title:      row.Title,
			Category:   row.Category,
			SolveCount: int(row.SolveCount),
			TotalTeams: int(row.TotalTeams),
			Percentage: pct,
		})
	}

	return out
}

func (r *StatisticsRepo) GetScoreDistribution(ctx context.Context) ([]*domain.ScoreDistributionBucket, error) {
	rows, err := r.Q(ctx).GetScoreDistribution(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreDistribution: %w", err)
	}

	out := make([]*domain.ScoreDistributionBucket, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ScoreDistributionBucket{
			Bucket: row.Bucket,
			Count:  int(row.Count),
		})
	}

	return out, nil
}

func (r *StatisticsRepo) GetScoreDistributionFrozen(ctx context.Context, freezeTime time.Time) ([]*domain.ScoreDistributionBucket, error) {
	rows, err := r.Q(ctx).GetScoreDistributionFrozen(ctx, pgutil.TimeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreDistributionFrozen: %w", err)
	}

	out := make([]*domain.ScoreDistributionBucket, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ScoreDistributionBucket{
			Bucket: row.Bucket,
			Count:  int(row.Count),
		})
	}

	return out, nil
}

func (r *StatisticsRepo) GetSubmissionTimeSeries(ctx context.Context) (*domain.SubmissionTimeSeriesStats, error) {
	rows, err := r.Q(ctx).GetSubmissionTimeSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSubmissionTimeSeries: %w", err)
	}

	items := make([]*domain.SubmissionTimeSeries, 0, len(rows))
	totalCorrect, totalIncorrect := 0, 0

	for _, row := range rows {
		dateStr := ""

		if row.Date.Valid {
			dateStr = row.Date.Time.Format(dateFormatISO)
		}

		items = append(items, &domain.SubmissionTimeSeries{
			Date:      dateStr,
			Correct:   int(row.Correct),
			Incorrect: int(row.Incorrect),
		})
		totalCorrect += int(row.Correct)
		totalIncorrect += int(row.Incorrect)
	}

	return &domain.SubmissionTimeSeriesStats{
		Items:          items,
		TotalCorrect:   totalCorrect,
		TotalIncorrect: totalIncorrect,
	}, nil
}

func (r *StatisticsRepo) GetSubmissionTimeSeriesFrozen(ctx context.Context, freezeTime time.Time) (*domain.SubmissionTimeSeriesStats, error) {
	rows, err := r.Q(ctx).GetSubmissionTimeSeriesFrozen(ctx, pgutil.TimeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSubmissionTimeSeriesFrozen: %w", err)
	}

	items := make([]*domain.SubmissionTimeSeries, 0, len(rows))
	totalCorrect, totalIncorrect := 0, 0

	for _, row := range rows {
		dateStr := ""

		if row.Date.Valid {
			dateStr = row.Date.Time.Format(dateFormatISO)
		}

		items = append(items, &domain.SubmissionTimeSeries{
			Date:      dateStr,
			Correct:   int(row.Correct),
			Incorrect: int(row.Incorrect),
		})
		totalCorrect += int(row.Correct)
		totalIncorrect += int(row.Incorrect)
	}

	return &domain.SubmissionTimeSeriesStats{
		Items:          items,
		TotalCorrect:   totalCorrect,
		TotalIncorrect: totalIncorrect,
	}, nil
}

func (r *StatisticsRepo) GetSubmissionTimeSeriesByType(ctx context.Context, isCorrect bool) ([]*domain.RegistrationTimePoint, error) {
	rows, err := r.Q(ctx).GetSubmissionTimeSeriesByType(ctx, isCorrect)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSubmissionTimeSeriesByType: %w", err)
	}

	return mapRegistrationTimePointRows(rows), nil
}

func (r *StatisticsRepo) GetSubmissionTimeSeriesByTypeFrozen(ctx context.Context, isCorrect bool, freezeTime time.Time) ([]*domain.RegistrationTimePoint, error) {
	rows, err := r.Q(ctx).GetSubmissionTimeSeriesByTypeFrozen(ctx, sqlc.GetSubmissionTimeSeriesByTypeFrozenParams{
		IsCorrect: isCorrect,
		CreatedAt: pgutil.TimeToTimestamptz(&freezeTime),
	})
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSubmissionTimeSeriesByTypeFrozen: %w", err)
	}

	out := make([]*domain.RegistrationTimePoint, 0, len(rows))
	for _, row := range rows {
		dateStr := ""

		if row.Date.Valid {
			dateStr = row.Date.Time.Format(time.DateOnly)
		}

		out = append(out, &domain.RegistrationTimePoint{
			Date:  dateStr,
			Count: int(row.Count),
		})
	}

	return out, nil
}

func mapRegistrationTimePointRows(rows []sqlc.GetSubmissionTimeSeriesByTypeRow) []*domain.RegistrationTimePoint {
	out := make([]*domain.RegistrationTimePoint, 0, len(rows))
	for _, row := range rows {
		dateStr := ""

		if row.Date.Valid {
			dateStr = row.Date.Time.Format(time.DateOnly)
		}

		out = append(out, &domain.RegistrationTimePoint{
			Date:  dateStr,
			Count: int(row.Count),
		})
	}

	return out
}

func (r *StatisticsRepo) GetTeamRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error) {
	rows, err := r.Q(ctx).GetTeamRegistrationTimeSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetTeamRegistrationTimeSeries: %w", err)
	}

	out := make([]*domain.RegistrationTimePoint, 0, len(rows))
	for _, row := range rows {
		dateStr := ""

		if row.Date.Valid {
			dateStr = row.Date.Time.Format(time.DateOnly)
		}

		out = append(out, &domain.RegistrationTimePoint{
			Date:  dateStr,
			Count: int(row.Count),
		})
	}

	return out, nil
}

func (r *StatisticsRepo) GetUserRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error) {
	rows, err := r.Q(ctx).GetUserRegistrationTimeSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetUserRegistrationTimeSeries: %w", err)
	}

	out := make([]*domain.RegistrationTimePoint, 0, len(rows))
	for _, row := range rows {
		dateStr := ""

		if row.Date.Valid {
			dateStr = row.Date.Time.Format(time.DateOnly)
		}

		out = append(out, &domain.RegistrationTimePoint{
			Date:  dateStr,
			Count: int(row.Count),
		})
	}

	return out, nil
}

func (r *StatisticsRepo) GetSolveMatrix(ctx context.Context) ([]*domain.SolveMatrixRow, error) {
	rows, err := r.Q(ctx).GetSolveMatrix(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSolveMatrix: %w", err)
	}

	out := make([]*domain.SolveMatrixRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.SolveMatrixRow{
			TeamID:            row.TeamID,
			TeamName:          row.TeamName,
			ChallengeID:       row.ChallengeID,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
			Solved:            row.Solved,
			SolvedAt:          pgutil.TimestamptzToTime(row.SolvedAt),
		})
	}

	return out, nil
}

func (r *StatisticsRepo) GetSolveMatrixFrozen(ctx context.Context, freezeTime time.Time) ([]*domain.SolveMatrixRow, error) {
	rows, err := r.Q(ctx).GetSolveMatrixFrozen(ctx, pgutil.TimeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSolveMatrixFrozen: %w", err)
	}

	out := make([]*domain.SolveMatrixRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.SolveMatrixRow{
			TeamID:            row.TeamID,
			TeamName:          row.TeamName,
			ChallengeID:       row.ChallengeID,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
			Solved:            row.Solved,
			SolvedAt:          pgutil.TimestamptzToTime(row.SolvedAt),
		})
	}

	return out, nil
}
