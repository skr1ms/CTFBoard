package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const dateFormatISO = "2006-01-02"

type StatisticsRepo struct {
	pool *pgxpool.Pool
}

var _ repo.StatisticsRepository = (*StatisticsRepo)(nil)

func NewStatisticsRepo(pool *pgxpool.Pool) *StatisticsRepo {
	return &StatisticsRepo{pool: pool}
}

func (r *StatisticsRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func (r *StatisticsRepo) GetGeneralStats(ctx context.Context) (*entity.GeneralStats, error) {
	var users, teams, challenges, solves int32

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		v, err := r.q(gCtx).CountUsers(gCtx)
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStats - CountUsers: %w", err)
		}
		users = v
		return nil
	})
	g.Go(func() error {
		v, err := r.q(gCtx).CountTeams(gCtx)
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStats - CountTeams: %w", err)
		}
		teams = v
		return nil
	})
	g.Go(func() error {
		v, err := r.q(gCtx).CountChallenges(gCtx)
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStats - CountChallenges: %w", err)
		}
		challenges = v
		return nil
	})
	g.Go(func() error {
		v, err := r.q(gCtx).CountSolves(gCtx)
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStats - CountSolves: %w", err)
		}
		solves = v
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetGeneralStats - Wait: %w", err)
	}
	return &entity.GeneralStats{
		UserCount:      int(users),
		TeamCount:      int(teams),
		ChallengeCount: int(challenges),
		SolveCount:     int(solves),
	}, nil
}

func (r *StatisticsRepo) GetGeneralStatsFrozen(ctx context.Context, freezeTime time.Time) (*entity.GeneralStats, error) {
	var users, teams, challenges, solves int32
	ft := &freezeTime

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		v, err := r.q(gCtx).CountUsersFrozen(gCtx, timeToTimestamptz(ft))
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStatsFrozen - CountUsersFrozen: %w", err)
		}
		users = v
		return nil
	})
	g.Go(func() error {
		v, err := r.q(gCtx).CountTeamsFrozen(gCtx, timeToTimestamptz(ft))
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStatsFrozen - CountTeamsFrozen: %w", err)
		}
		teams = v
		return nil
	})
	g.Go(func() error {
		v, err := r.q(gCtx).CountChallenges(gCtx)
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStatsFrozen - CountChallenges: %w", err)
		}
		challenges = v
		return nil
	})
	g.Go(func() error {
		v, err := r.q(gCtx).CountSolvesFrozen(gCtx, timeToTimestamptz(&freezeTime))
		if err != nil {
			return fmt.Errorf("StatisticsRepo - GetGeneralStatsFrozen - CountSolvesFrozen: %w", err)
		}
		solves = v
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetGeneralStatsFrozen - Wait: %w", err)
	}
	return &entity.GeneralStats{
		UserCount:      int(users),
		TeamCount:      int(teams),
		ChallengeCount: int(challenges),
		SolveCount:     int(solves),
	}, nil
}

func (r *StatisticsRepo) GetChallengeStats(ctx context.Context) ([]*entity.ChallengeStats, error) {
	rows, err := r.q(ctx).GetChallengeStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeStats: %w", err)
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

func (r *StatisticsRepo) GetChallengeStatsFrozen(ctx context.Context, freezeTime time.Time) ([]*entity.ChallengeStats, error) {
	rows, err := r.q(ctx).GetChallengeStatsFrozen(ctx, timeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeStatsFrozen: %w", err)
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

func (r *StatisticsRepo) GetChallengeDetailStats(ctx context.Context, challengeID uuid.UUID) (*entity.ChallengeDetailStats, error) {
	chRow, err := r.q(ctx).GetChallengeDetailChallenge(ctx, challengeID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeDetailStats - Challenge: %w", err)
	}
	percentageSolved := 0.0
	if chRow.TotalTeams > 0 {
		percentageSolved = float64(chRow.SolveCount) / float64(chRow.TotalTeams) * 100
	}
	solveRows, err := r.q(ctx).GetChallengeDetailSolves(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeDetailStats - Solves: %w", err)
	}
	solves := make([]entity.ChallengeSolveEntry, 0, len(solveRows))
	for _, row := range solveRows {
		solves = append(solves, entity.ChallengeSolveEntry{
			TeamID:   row.TeamID,
			TeamName: row.TeamName,
			SolvedAt: ptrTimeToTime(timestamptzToTime(row.SolvedAt)),
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

func (r *StatisticsRepo) GetChallengeDetailStatsFrozen(ctx context.Context, challengeID uuid.UUID, freezeTime time.Time) (*entity.ChallengeDetailStats, error) {
	chRow, err := r.q(ctx).GetChallengeDetailChallengeFrozen(ctx, sqlc.GetChallengeDetailChallengeFrozenParams{
		ID:       challengeID,
		SolvedAt: timeToTimestamptz(&freezeTime),
	})
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeDetailStatsFrozen - Challenge: %w", err)
	}
	percentageSolved := 0.0
	if chRow.TotalTeams > 0 {
		percentageSolved = float64(chRow.SolveCount) / float64(chRow.TotalTeams) * 100
	}
	solveRows, err := r.q(ctx).GetChallengeDetailSolvesFrozen(ctx, sqlc.GetChallengeDetailSolvesFrozenParams{
		ChallengeID: challengeID,
		SolvedAt:    timeToTimestamptz(&freezeTime),
	})
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeDetailStatsFrozen - Solves: %w", err)
	}
	solves := make([]entity.ChallengeSolveEntry, 0, len(solveRows))
	for _, row := range solveRows {
		solves = append(solves, entity.ChallengeSolveEntry{
			TeamID:   row.TeamID,
			TeamName: row.TeamName,
			SolvedAt: ptrTimeToTime(timestamptzToTime(row.SolvedAt)),
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

func (r *StatisticsRepo) GetScoreboardHistory(ctx context.Context, limit int) ([]*entity.ScoreboardHistoryEntry, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreboardHistory - limit: %w", err)
	}
	rows, err := r.q(ctx).GetScoreboardHistory(ctx, limit32)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreboardHistory: %w", err)
	}
	return mapScoreboardHistoryRows(rows), nil
}

func (r *StatisticsRepo) GetScoreboardHistoryFrozen(ctx context.Context, freezeTime time.Time, limit int) ([]*entity.ScoreboardHistoryEntry, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreboardHistoryFrozen - limit: %w", err)
	}
	rows, err := r.q(ctx).GetScoreboardHistoryFrozen(ctx, sqlc.GetScoreboardHistoryFrozenParams{
		Limit:    limit32,
		SolvedAt: timeToTimestamptz(&freezeTime),
	})
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreboardHistoryFrozen: %w", err)
	}
	out := make([]*entity.ScoreboardHistoryEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.ScoreboardHistoryEntry{
			TeamID:    row.TeamID,
			TeamName:  row.TeamName,
			Points:    int(row.Points),
			Timestamp: ptrTimeToTime(timestamptzToTime(row.Timestamp)),
		})
	}
	return out, nil
}

func mapScoreboardHistoryRows(rows []sqlc.GetScoreboardHistoryRow) []*entity.ScoreboardHistoryEntry {
	out := make([]*entity.ScoreboardHistoryEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.ScoreboardHistoryEntry{
			TeamID:    row.TeamID,
			TeamName:  row.TeamName,
			Points:    int(row.Points),
			Timestamp: ptrTimeToTime(timestamptzToTime(row.Timestamp)),
		})
	}
	return out
}

func (r *StatisticsRepo) GetChallengeSolvePercentages(ctx context.Context) ([]*entity.ChallengeSolvePercentage, error) {
	rows, err := r.q(ctx).GetChallengeSolvePercentages(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeSolvePercentages: %w", err)
	}
	return mapChallengeSolvePercentageRows(rows), nil
}

func (r *StatisticsRepo) GetChallengeSolvePercentagesFrozen(ctx context.Context, freezeTime time.Time) ([]*entity.ChallengeSolvePercentage, error) {
	rows, err := r.q(ctx).GetChallengeSolvePercentagesFrozen(ctx, timeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetChallengeSolvePercentagesFrozen: %w", err)
	}
	out := make([]*entity.ChallengeSolvePercentage, 0, len(rows))
	for _, row := range rows {
		pct := 0.0
		if row.Percentage != nil {
			if v, ok := row.Percentage.(float64); ok {
				pct = v
			}
		}
		out = append(out, &entity.ChallengeSolvePercentage{
			ID:         row.ID,
			Title:      row.Title,
			Category:   ptrStrToStr(row.Category),
			SolveCount: int(row.SolveCount),
			TotalTeams: int(row.TotalTeams),
			Percentage: pct,
		})
	}
	return out, nil
}

func mapChallengeSolvePercentageRows(rows []sqlc.GetChallengeSolvePercentagesRow) []*entity.ChallengeSolvePercentage {
	out := make([]*entity.ChallengeSolvePercentage, 0, len(rows))
	for _, row := range rows {
		pct := 0.0
		if row.Percentage != nil {
			if v, ok := row.Percentage.(float64); ok {
				pct = v
			}
		}
		out = append(out, &entity.ChallengeSolvePercentage{
			ID:         row.ID,
			Title:      row.Title,
			Category:   ptrStrToStr(row.Category),
			SolveCount: int(row.SolveCount),
			TotalTeams: int(row.TotalTeams),
			Percentage: pct,
		})
	}
	return out
}

func (r *StatisticsRepo) GetScoreDistribution(ctx context.Context) ([]*entity.ScoreDistributionBucket, error) {
	rows, err := r.q(ctx).GetScoreDistribution(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreDistribution: %w", err)
	}
	out := make([]*entity.ScoreDistributionBucket, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.ScoreDistributionBucket{
			Bucket: row.Bucket,
			Count:  int(row.Count),
		})
	}
	return out, nil
}

func (r *StatisticsRepo) GetScoreDistributionFrozen(ctx context.Context, freezeTime time.Time) ([]*entity.ScoreDistributionBucket, error) {
	rows, err := r.q(ctx).GetScoreDistributionFrozen(ctx, timeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetScoreDistributionFrozen: %w", err)
	}
	out := make([]*entity.ScoreDistributionBucket, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.ScoreDistributionBucket{
			Bucket: row.Bucket,
			Count:  int(row.Count),
		})
	}
	return out, nil
}

func (r *StatisticsRepo) GetSubmissionTimeSeries(ctx context.Context) (*entity.SubmissionTimeSeriesStats, error) {
	rows, err := r.q(ctx).GetSubmissionTimeSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSubmissionTimeSeries: %w", err)
	}
	items := make([]*entity.SubmissionTimeSeries, 0, len(rows))
	totalCorrect, totalIncorrect := 0, 0
	for _, row := range rows {
		dateStr := ""
		if row.Date.Valid {
			dateStr = row.Date.Time.Format(dateFormatISO)
		}
		items = append(items, &entity.SubmissionTimeSeries{
			Date:      dateStr,
			Correct:   int(row.Correct),
			Incorrect: int(row.Incorrect),
		})
		totalCorrect += int(row.Correct)
		totalIncorrect += int(row.Incorrect)
	}
	return &entity.SubmissionTimeSeriesStats{
		Items:          items,
		TotalCorrect:   totalCorrect,
		TotalIncorrect: totalIncorrect,
	}, nil
}

func (r *StatisticsRepo) GetSubmissionTimeSeriesFrozen(ctx context.Context, freezeTime time.Time) (*entity.SubmissionTimeSeriesStats, error) {
	rows, err := r.q(ctx).GetSubmissionTimeSeriesFrozen(ctx, timeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSubmissionTimeSeriesFrozen: %w", err)
	}
	items := make([]*entity.SubmissionTimeSeries, 0, len(rows))
	totalCorrect, totalIncorrect := 0, 0
	for _, row := range rows {
		dateStr := ""
		if row.Date.Valid {
			dateStr = row.Date.Time.Format(dateFormatISO)
		}
		items = append(items, &entity.SubmissionTimeSeries{
			Date:      dateStr,
			Correct:   int(row.Correct),
			Incorrect: int(row.Incorrect),
		})
		totalCorrect += int(row.Correct)
		totalIncorrect += int(row.Incorrect)
	}
	return &entity.SubmissionTimeSeriesStats{
		Items:          items,
		TotalCorrect:   totalCorrect,
		TotalIncorrect: totalIncorrect,
	}, nil
}

func (r *StatisticsRepo) GetSubmissionTimeSeriesByType(ctx context.Context, isCorrect bool) ([]*entity.RegistrationTimePoint, error) {
	rows, err := r.q(ctx).GetSubmissionTimeSeriesByType(ctx, isCorrect)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSubmissionTimeSeriesByType: %w", err)
	}
	return mapRegistrationTimePointRows(rows), nil
}

func (r *StatisticsRepo) GetSubmissionTimeSeriesByTypeFrozen(ctx context.Context, isCorrect bool, freezeTime time.Time) ([]*entity.RegistrationTimePoint, error) {
	rows, err := r.q(ctx).GetSubmissionTimeSeriesByTypeFrozen(ctx, sqlc.GetSubmissionTimeSeriesByTypeFrozenParams{
		IsCorrect: isCorrect,
		CreatedAt: timeToTimestamptz(&freezeTime),
	})
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSubmissionTimeSeriesByTypeFrozen: %w", err)
	}
	out := make([]*entity.RegistrationTimePoint, 0, len(rows))
	for _, row := range rows {
		dateStr := ""
		if row.Date.Valid {
			dateStr = row.Date.Time.Format("2006-01-02")
		}
		out = append(out, &entity.RegistrationTimePoint{
			Date:  dateStr,
			Count: int(row.Count),
		})
	}
	return out, nil
}

func mapRegistrationTimePointRows(rows []sqlc.GetSubmissionTimeSeriesByTypeRow) []*entity.RegistrationTimePoint {
	out := make([]*entity.RegistrationTimePoint, 0, len(rows))
	for _, row := range rows {
		dateStr := ""
		if row.Date.Valid {
			dateStr = row.Date.Time.Format("2006-01-02")
		}
		out = append(out, &entity.RegistrationTimePoint{
			Date:  dateStr,
			Count: int(row.Count),
		})
	}
	return out
}

func (r *StatisticsRepo) GetTeamRegistrationTimeSeries(ctx context.Context) ([]*entity.RegistrationTimePoint, error) {
	rows, err := r.q(ctx).GetTeamRegistrationTimeSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetTeamRegistrationTimeSeries: %w", err)
	}
	out := make([]*entity.RegistrationTimePoint, 0, len(rows))
	for _, row := range rows {
		dateStr := ""
		if row.Date.Valid {
			dateStr = row.Date.Time.Format("2006-01-02")
		}
		out = append(out, &entity.RegistrationTimePoint{
			Date:  dateStr,
			Count: int(row.Count),
		})
	}
	return out, nil
}

func (r *StatisticsRepo) GetUserRegistrationTimeSeries(ctx context.Context) ([]*entity.RegistrationTimePoint, error) {
	rows, err := r.q(ctx).GetUserRegistrationTimeSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetUserRegistrationTimeSeries: %w", err)
	}
	out := make([]*entity.RegistrationTimePoint, 0, len(rows))
	for _, row := range rows {
		dateStr := ""
		if row.Date.Valid {
			dateStr = row.Date.Time.Format("2006-01-02")
		}
		out = append(out, &entity.RegistrationTimePoint{
			Date:  dateStr,
			Count: int(row.Count),
		})
	}
	return out, nil
}

func (r *StatisticsRepo) GetSolveMatrix(ctx context.Context) ([]*entity.SolveMatrixRow, error) {
	rows, err := r.q(ctx).GetSolveMatrix(ctx)
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSolveMatrix: %w", err)
	}
	out := make([]*entity.SolveMatrixRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.SolveMatrixRow{
			TeamID:            row.TeamID,
			TeamName:          row.TeamName,
			ChallengeID:       row.ChallengeID,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: ptrStrToStr(row.ChallengeCategory),
			Solved:            row.Solved,
			SolvedAt:          timestamptzToTime(row.SolvedAt),
		})
	}
	return out, nil
}

func (r *StatisticsRepo) GetSolveMatrixFrozen(ctx context.Context, freezeTime time.Time) ([]*entity.SolveMatrixRow, error) {
	rows, err := r.q(ctx).GetSolveMatrixFrozen(ctx, timeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("StatisticsRepo - GetSolveMatrixFrozen: %w", err)
	}
	out := make([]*entity.SolveMatrixRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.SolveMatrixRow{
			TeamID:            row.TeamID,
			TeamName:          row.TeamName,
			ChallengeID:       row.ChallengeID,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: ptrStrToStr(row.ChallengeCategory),
			Solved:            row.Solved,
			SolvedAt:          timestamptzToTime(row.SolvedAt),
		})
	}
	return out, nil
}
