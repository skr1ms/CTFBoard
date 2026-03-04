package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SolveRepo struct {
	pool *pgxpool.Pool
}

var _ repo.SolveRepository = (*SolveRepo)(nil)

func NewSolveRepo(pool *pgxpool.Pool) *SolveRepo {
	return &SolveRepo{pool: pool}
}

func (r *SolveRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func toEntitySolve(s sqlc.Solve) *entity.Solve {
	return &entity.Solve{
		ID:            s.ID,
		UserID:        s.UserID,
		TeamID:        s.TeamID,
		ChallengeID:   s.ChallengeID,
		SolvedAt:      ptrTimeToTime(s.SolvedAt),
		PointsAtSolve: int(s.PointsAtSolve),
	}
}

func toScoreboardEntry(row sqlc.GetScoreboardRow) *repo.ScoreboardEntry {
	return &repo.ScoreboardEntry{
		TeamID:   row.TeamID,
		TeamName: row.TeamName,
		Points:   int(row.Points),
		SolvedAt: timeFromNullableAny(row.SolvedAt),
	}
}

func toScoreboardEntryFrozen(row sqlc.GetScoreboardFrozenRow) *repo.ScoreboardEntry {
	return &repo.ScoreboardEntry{
		TeamID:   row.TeamID,
		TeamName: row.TeamName,
		Points:   int(row.Points),
		SolvedAt: timeFromNullableAny(row.SolvedAt),
	}
}

func toScoreboardEntryByBracket(row sqlc.GetScoreboardByBracketRow) *repo.ScoreboardEntry {
	return &repo.ScoreboardEntry{
		TeamID:   row.TeamID,
		TeamName: row.TeamName,
		Points:   int(row.Points),
		SolvedAt: timeFromNullableAny(row.SolvedAt),
	}
}

func toScoreboardEntryByBracketFrozen(row sqlc.GetScoreboardByBracketFrozenRow) *repo.ScoreboardEntry {
	return &repo.ScoreboardEntry{
		TeamID:   row.TeamID,
		TeamName: row.TeamName,
		Points:   int(row.Points),
		SolvedAt: timeFromNullableAny(row.SolvedAt),
	}
}

func toFirstBloodEntry(row sqlc.GetFirstBloodRow) *repo.FirstBloodEntry {
	return &repo.FirstBloodEntry{
		UserID:   row.UserID,
		Username: row.Username,
		TeamID:   row.TeamID,
		TeamName: row.TeamName,
		SolvedAt: ptrTimeToTime(row.SolvedAt),
	}
}

func (r *SolveRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Solve, error) {
	s, err := r.q(ctx).GetSolveByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByID: %w", err)
	}
	return toEntitySolve(s), nil
}

func (r *SolveRepo) GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*entity.Solve, error) {
	s, err := r.q(ctx).GetSolveByTeamAndChallenge(ctx, sqlc.GetSolveByTeamAndChallengeParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallenge: %w", err)
	}
	return toEntitySolve(s), nil
}

func (r *SolveRepo) GetSolvedChallengeIDsByTeam(ctx context.Context, teamID uuid.UUID, challengeIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(challengeIDs) == 0 {
		return nil, nil
	}
	ids, err := r.q(ctx).GetSolvedChallengeIDsByTeam(ctx, sqlc.GetSolvedChallengeIDsByTeamParams{
		TeamID:  teamID,
		Column2: challengeIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetSolvedChallengeIDsByTeam: %w", err)
	}
	return ids, nil
}

func (r *SolveRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Solve, error) {
	rows, err := r.q(ctx).GetSolvesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserID: %w", err)
	}
	out := make([]*entity.Solve, 0, len(rows))
	for _, s := range rows {
		out = append(out, toEntitySolve(s))
	}
	return out, nil
}

func (r *SolveRepo) GetScoreboard(ctx context.Context) ([]*repo.ScoreboardEntry, error) {
	rows, err := r.q(ctx).GetScoreboard(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboard: %w", err)
	}
	out := make([]*repo.ScoreboardEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toScoreboardEntry(row))
	}
	return out, nil
}

func (r *SolveRepo) GetScoreboardFrozen(ctx context.Context, freezeTime time.Time) ([]*repo.ScoreboardEntry, error) {
	rows, err := r.q(ctx).GetScoreboardFrozen(ctx, sqlc.GetScoreboardFrozenParams{
		SolvedAt:  &freezeTime,
		CreatedAt: &freezeTime,
	})
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboardFrozen: %w", err)
	}
	out := make([]*repo.ScoreboardEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toScoreboardEntryFrozen(row))
	}
	return out, nil
}

func (r *SolveRepo) GetScoreboardByBracket(ctx context.Context, bracketID *uuid.UUID) ([]*repo.ScoreboardEntry, error) {
	rows, err := r.q(ctx).GetScoreboardByBracket(ctx, bracketID)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboardByBracket: %w", err)
	}
	out := make([]*repo.ScoreboardEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toScoreboardEntryByBracket(row))
	}
	return out, nil
}

func (r *SolveRepo) GetScoreboardByBracketFrozen(ctx context.Context, freezeTime time.Time, bracketID *uuid.UUID) ([]*repo.ScoreboardEntry, error) {
	rows, err := r.q(ctx).GetScoreboardByBracketFrozen(ctx, sqlc.GetScoreboardByBracketFrozenParams{
		SolvedAt:  &freezeTime,
		CreatedAt: &freezeTime,
		BracketID: bracketID,
	})
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetScoreboardByBracketFrozen: %w", err)
	}
	out := make([]*repo.ScoreboardEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toScoreboardEntryByBracketFrozen(row))
	}
	return out, nil
}

// GetTeamScore returns the team's real-time total (solves + awards), deliberately
// ignoring freeze time. Teams see their actual score during the frozen period;
// the public scoreboard is what is frozen, not the team's own view.
func (r *SolveRepo) GetTeamScore(ctx context.Context, teamID uuid.UUID) (int, error) {
	total, err := r.q(ctx).GetTeamScore(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("SolveRepo - GetTeamScore: %w", err)
	}
	return int(total), nil
}

func (r *SolveRepo) GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*repo.FirstBloodEntry, error) {
	row, err := r.q(ctx).GetFirstBlood(ctx, challengeID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetFirstBlood: %w", err)
	}
	return toFirstBloodEntry(row), nil
}

func (r *SolveRepo) GetAll(ctx context.Context) ([]*entity.Solve, error) {
	rows, err := r.q(ctx).GetAllSolves(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetAll: %w", err)
	}
	out := make([]*entity.Solve, 0, len(rows))
	for _, s := range rows {
		out = append(out, toEntitySolve(s))
	}
	return out, nil
}

func (r *SolveRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.SolveWithDetails, error) {
	rows, err := r.q(ctx).GetSolvesByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByChallengeID: %w", err)
	}
	out := make([]*entity.SolveWithDetails, 0, len(rows))
	for _, s := range rows {
		out = append(out, &entity.SolveWithDetails{
			Solve: entity.Solve{
				ID:          s.ID,
				UserID:      s.UserID,
				TeamID:      s.TeamID,
				ChallengeID: s.ChallengeID,
				SolvedAt:    ptrTimeToTime(s.SolvedAt),
			},
			Username: s.Username,
			TeamName: s.TeamName,
		})
	}
	return out, nil
}

func (r *SolveRepo) GetByUserIDWithDetails(ctx context.Context, userID uuid.UUID) ([]*entity.SolveWithDetails, error) {
	rows, err := r.q(ctx).GetSolvesByUserIDWithDetails(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserIDWithDetails: %w", err)
	}
	out := make([]*entity.SolveWithDetails, 0, len(rows))
	for _, s := range rows {
		out = append(out, &entity.SolveWithDetails{
			Solve: entity.Solve{
				ID:          s.ID,
				UserID:      s.UserID,
				TeamID:      s.TeamID,
				ChallengeID: s.ChallengeID,
				SolvedAt:    ptrTimeToTime(s.SolvedAt),
			},
			ChallengeTitle:    s.ChallengeTitle,
			ChallengeCategory: ptrStrToStr(s.ChallengeCategory),
			ChallengePoints:   int32PtrToInt(s.ChallengePoints),
		})
	}
	return out, nil
}

func (r *SolveRepo) GetByTeamIDWithDetails(ctx context.Context, teamID uuid.UUID) ([]*entity.SolveWithDetails, error) {
	rows, err := r.q(ctx).GetSolvesByTeamIDWithDetails(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByTeamIDWithDetails: %w", err)
	}
	out := make([]*entity.SolveWithDetails, 0, len(rows))
	for _, s := range rows {
		out = append(out, &entity.SolveWithDetails{
			Solve: entity.Solve{
				ID:          s.ID,
				UserID:      s.UserID,
				TeamID:      s.TeamID,
				ChallengeID: s.ChallengeID,
				SolvedAt:    ptrTimeToTime(s.SolvedAt),
			},
			Username:          s.Username,
			ChallengeTitle:    s.ChallengeTitle,
			ChallengeCategory: ptrStrToStr(s.ChallengeCategory),
			ChallengePoints:   int32PtrToInt(s.ChallengePoints),
		})
	}
	return out, nil
}

func (r *SolveRepo) Create(ctx context.Context, s *entity.Solve) error {
	s.ID = uuid.New()
	s.SolvedAt = time.Now()
	err := r.q(ctx).CreateSolve(ctx, sqlc.CreateSolveParams{
		ID:            s.ID,
		UserID:        s.UserID,
		TeamID:        s.TeamID,
		ChallengeID:   s.ChallengeID,
		SolvedAt:      &s.SolvedAt,
		PointsAtSolve: int32(s.PointsAtSolve), //nolint:gosec // points are capped by scoring config; no realistic overflow
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return httperr.ErrAlreadySolved
		}
		return fmt.Errorf("SolveRepo - Create: %w", err)
	}
	return nil
}

func (r *SolveRepo) GetByTeamAndChallengeForUpdate(ctx context.Context, teamID, challengeID uuid.UUID) (*entity.Solve, error) {
	s, err := r.q(ctx).GetSolveByTeamAndChallengeForUpdate(ctx, sqlc.GetSolveByTeamAndChallengeForUpdateParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallengeForUpdate: %w", err)
	}
	return toEntitySolve(s), nil
}

// DeleteByTeamAndChallenge removes a solve by team and challenge. Idempotent: returns nil if the solve does not exist.
func (r *SolveRepo) DeleteByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) error {
	if err := r.q(ctx).DeleteSolveByTeamAndChallenge(ctx, sqlc.DeleteSolveByTeamAndChallengeParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	}); err != nil {
		return fmt.Errorf("SolveRepo - DeleteByTeamAndChallenge: %w", err)
	}
	return nil
}

func (r *SolveRepo) DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error {
	if err := r.q(ctx).DeleteSolvesByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("SolveRepo - DeleteByTeamID: %w", err)
	}
	return nil
}
