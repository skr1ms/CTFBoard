package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"

	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type SolveRepo struct {
	BaseRepo
}

var _ repo.SolveRepository = (*SolveRepo)(nil)

func NewSolveRepo(pool *pgxpool.Pool) *SolveRepo {
	return &SolveRepo{BaseRepo: BaseRepo{pool: pool}}
}

func toDomainSolve(s sqlc.Solve) *domain.Solve {
	return &domain.Solve{
		ID:            s.ID,
		UserID:        s.UserID,
		TeamID:        s.TeamID,
		ChallengeID:   s.ChallengeID,
		SolvedAt:      pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(s.SolvedAt)),
		PointsAtSolve: int(s.PointsAtSolve),
		BannedTeamID:  s.BannedTeamID,
		BannedUserID:  s.BannedUserID,
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
		SolvedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.SolvedAt)),
	}
}

func (r *SolveRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Solve, error) {
	s, err := r.Q(ctx).GetSolveByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByID: %w", err)
	}
	return toDomainSolve(s), nil
}

func (r *SolveRepo) GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*domain.Solve, error) {
	s, err := r.Q(ctx).GetSolveByTeamAndChallenge(ctx, sqlc.GetSolveByTeamAndChallengeParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallenge: %w", err)
	}
	return toDomainSolve(s), nil
}

func (r *SolveRepo) GetSolvedChallengeIDsByTeam(ctx context.Context, teamID uuid.UUID, challengeIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(challengeIDs) == 0 {
		return nil, nil
	}
	ids, err := r.Q(ctx).GetSolvedChallengeIDsByTeam(ctx, sqlc.GetSolvedChallengeIDsByTeamParams{
		TeamID:  teamID,
		Column2: challengeIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetSolvedChallengeIDsByTeam: %w", err)
	}
	return ids, nil
}

func (r *SolveRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Solve, error) {
	rows, err := r.Q(ctx).GetSolvesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserID: %w", err)
	}
	out := make([]*domain.Solve, 0, len(rows))
	for _, s := range rows {
		out = append(out, toDomainSolve(s))
	}
	return out, nil
}

func (r *SolveRepo) GetScoreboard(ctx context.Context) ([]*repo.ScoreboardEntry, error) {
	rows, err := r.Q(ctx).GetScoreboard(ctx)
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
	return r.GetScoreboardByBracketFrozen(ctx, freezeTime, nil)
}

func (r *SolveRepo) GetScoreboardByBracket(ctx context.Context, bracketID *uuid.UUID) ([]*repo.ScoreboardEntry, error) {
	rows, err := r.Q(ctx).GetScoreboardByBracket(ctx, bracketID)
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
	rows, err := r.Q(ctx).GetScoreboardByBracketFrozen(ctx, sqlc.GetScoreboardByBracketFrozenParams{
		SolvedAt:  pgutil.TimeToTimestamptz(&freezeTime),
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
	total, err := r.Q(ctx).GetTeamScore(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("SolveRepo - GetTeamScore: %w", err)
	}
	return int(total), nil
}

func (r *SolveRepo) GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*repo.FirstBloodEntry, error) {
	row, err := r.Q(ctx).GetFirstBlood(ctx, challengeID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetFirstBlood: %w", err)
	}
	return toFirstBloodEntry(row), nil
}

func (r *SolveRepo) GetFirstBloodFrozen(ctx context.Context, challengeID uuid.UUID, freezeTime time.Time) (*repo.FirstBloodEntry, error) {
	row, err := r.Q(ctx).GetFirstBloodFrozen(ctx, sqlc.GetFirstBloodFrozenParams{
		ChallengeID: challengeID,
		SolvedAt:    pgutil.TimeToTimestamptz(&freezeTime),
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetFirstBloodFrozen: %w", err)
	}
	return &repo.FirstBloodEntry{
		UserID:   row.UserID,
		Username: row.Username,
		TeamID:   row.TeamID,
		TeamName: row.TeamName,
		SolvedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.SolvedAt)),
	}, nil
}

func (r *SolveRepo) GetAll(ctx context.Context) ([]*domain.Solve, error) {
	rows, err := r.Q(ctx).GetAllSolves(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetAll: %w", err)
	}
	out := make([]*domain.Solve, 0, len(rows))
	for _, s := range rows {
		out = append(out, toDomainSolve(s))
	}
	return out, nil
}

func (r *SolveRepo) GetAllForBackup(ctx context.Context) ([]*domain.Solve, error) {
	rows, err := r.Q(ctx).GetSolvesForBackup(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetAllForBackup: %w", err)
	}
	out := make([]*domain.Solve, 0, len(rows))
	for _, s := range rows {
		out = append(out, toDomainSolve(s))
	}
	return out, nil
}

func (r *SolveRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.SolveWithDetails, error) {
	rows, err := r.Q(ctx).GetSolvesByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByChallengeID: %w", err)
	}
	out := make([]*domain.SolveWithDetails, 0, len(rows))
	for _, s := range rows {
		out = append(out, &domain.SolveWithDetails{
			Solve: domain.Solve{
				ID:          s.ID,
				UserID:      s.UserID,
				TeamID:      s.TeamID,
				ChallengeID: s.ChallengeID,
				SolvedAt:    pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(s.SolvedAt)),
			},
			Username: s.Username,
			TeamName: s.TeamName,
		})
	}
	return out, nil
}

func (r *SolveRepo) GetByChallengeIDFrozen(ctx context.Context, challengeID uuid.UUID, freezeTime time.Time) ([]*domain.SolveWithDetails, error) {
	rows, err := r.Q(ctx).GetSolvesByChallengeIDFrozen(ctx, sqlc.GetSolvesByChallengeIDFrozenParams{
		ChallengeID: challengeID,
		SolvedAt:    pgutil.TimeToTimestamptz(&freezeTime),
	})
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByChallengeIDFrozen: %w", err)
	}
	out := make([]*domain.SolveWithDetails, 0, len(rows))
	for _, s := range rows {
		out = append(out, &domain.SolveWithDetails{
			Solve: domain.Solve{
				ID:          s.ID,
				UserID:      s.UserID,
				TeamID:      s.TeamID,
				ChallengeID: s.ChallengeID,
				SolvedAt:    pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(s.SolvedAt)),
			},
			Username: s.Username,
			TeamName: s.TeamName,
		})
	}
	return out, nil
}

func (r *SolveRepo) GetSolveCountsFrozen(ctx context.Context, freezeTime time.Time) (map[uuid.UUID]int, error) {
	rows, err := r.Q(ctx).GetSolveCountsFrozen(ctx, pgutil.TimeToTimestamptz(&freezeTime))
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetSolveCountsFrozen: %w", err)
	}
	out := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		out[row.ChallengeID] = int(row.SolveCount)
	}
	return out, nil
}

func (r *SolveRepo) GetByUserIDWithDetails(ctx context.Context, userID uuid.UUID) ([]*domain.SolveWithDetails, error) {
	rows, err := r.Q(ctx).GetSolvesByUserIDWithDetails(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByUserIDWithDetails: %w", err)
	}
	out := make([]*domain.SolveWithDetails, 0, len(rows))
	for _, s := range rows {
		out = append(out, &domain.SolveWithDetails{
			Solve: domain.Solve{
				ID:          s.ID,
				UserID:      s.UserID,
				TeamID:      s.TeamID,
				ChallengeID: s.ChallengeID,
				SolvedAt:    pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(s.SolvedAt)),
			},
			ChallengeTitle:    s.ChallengeTitle,
			ChallengeCategory: s.ChallengeCategory,
			ChallengePoints:   int(lo.FromPtr(s.ChallengePoints)),
		})
	}
	return out, nil
}

func (r *SolveRepo) GetByTeamIDWithDetails(ctx context.Context, teamID uuid.UUID) ([]*domain.SolveWithDetails, error) {
	rows, err := r.Q(ctx).GetSolvesByTeamIDWithDetails(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetByTeamIDWithDetails: %w", err)
	}
	out := make([]*domain.SolveWithDetails, 0, len(rows))
	for _, s := range rows {
		out = append(out, &domain.SolveWithDetails{
			Solve: domain.Solve{
				ID:          s.ID,
				UserID:      s.UserID,
				TeamID:      s.TeamID,
				ChallengeID: s.ChallengeID,
				SolvedAt:    pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(s.SolvedAt)),
			},
			Username:          s.Username,
			ChallengeTitle:    s.ChallengeTitle,
			ChallengeCategory: s.ChallengeCategory,
			ChallengePoints:   int(lo.FromPtr(s.ChallengePoints)),
		})
	}
	return out, nil
}

func (r *SolveRepo) Create(ctx context.Context, s *domain.Solve) error {
	s.ID = uuid.New()
	s.SolvedAt = time.Now()
	err := r.Q(ctx).CreateSolve(ctx, sqlc.CreateSolveParams{
		ID:            s.ID,
		UserID:        s.UserID,
		TeamID:        s.TeamID,
		ChallengeID:   s.ChallengeID,
		SolvedAt:      pgutil.TimeToTimestamptz(&s.SolvedAt),
		PointsAtSolve: int32(s.PointsAtSolve), //nolint:gosec // points are capped by scoring config; no realistic overflow
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return httperr.ErrAlreadySolved
		}
		return fmt.Errorf("SolveRepo - Create: %w", err)
	}
	return nil
}

func (r *SolveRepo) GetByTeamAndChallengeForUpdate(ctx context.Context, teamID, challengeID uuid.UUID) (*domain.Solve, error) {
	s, err := r.Q(ctx).GetSolveByTeamAndChallengeForUpdate(ctx, sqlc.GetSolveByTeamAndChallengeForUpdateParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrSolveNotFound
		}
		return nil, fmt.Errorf("SolveRepo - GetByTeamAndChallengeForUpdate: %w", err)
	}
	return toDomainSolve(s), nil
}

// DeleteByTeamAndChallenge removes a solve by team and challenge. Idempotent: returns nil if the solve does not exist.
func (r *SolveRepo) DeleteByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) error {
	if err := r.Q(ctx).DeleteSolveByTeamAndChallenge(ctx, sqlc.DeleteSolveByTeamAndChallengeParams{
		TeamID:      teamID,
		ChallengeID: challengeID,
	}); err != nil {
		return fmt.Errorf("SolveRepo - DeleteByTeamAndChallenge: %w", err)
	}
	return nil
}

func (r *SolveRepo) DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error {
	if err := r.Q(ctx).DeleteSolvesByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("SolveRepo - DeleteByTeamID: %w", err)
	}
	return nil
}

func (r *SolveRepo) SoftBanByTeamID(ctx context.Context, teamID uuid.UUID) error {
	if err := r.Q(ctx).SoftBanSolvesByTeamID(ctx, teamID); err != nil {
		return fmt.Errorf("SolveRepo - SoftBanByTeamID: %w", err)
	}
	return nil
}

func (r *SolveRepo) RestoreByBannedTeamID(ctx context.Context, teamID uuid.UUID) error {
	if err := r.Q(ctx).RestoreSolvesByBannedTeamID(ctx, &teamID); err != nil {
		return fmt.Errorf("SolveRepo - RestoreByBannedTeamID: %w", err)
	}
	return nil
}

func (r *SolveRepo) SoftBanByTeamIDAndUserID(ctx context.Context, teamID, userID uuid.UUID) error {
	if err := r.Q(ctx).SoftBanSolvesByTeamIDAndUserID(ctx, sqlc.SoftBanSolvesByTeamIDAndUserIDParams{
		TeamID: teamID,
		UserID: userID,
	}); err != nil {
		return fmt.Errorf("SolveRepo - SoftBanByTeamIDAndUserID: %w", err)
	}
	return nil
}

func (r *SolveRepo) RestoreByBannedUserID(ctx context.Context, userID uuid.UUID) error {
	if err := r.Q(ctx).RestoreSolvesByBannedUserID(ctx, &userID); err != nil {
		return fmt.Errorf("SolveRepo - RestoreByBannedUserID: %w", err)
	}
	return nil
}

func (r *SolveRepo) GetSolvesForPointsRecalc(ctx context.Context, challengeIDs []uuid.UUID) ([]*repo.SolveForPointsRecalc, error) {
	if len(challengeIDs) == 0 {
		return nil, nil
	}
	rows, err := r.Q(ctx).GetSolvesForPointsRecalc(ctx, challengeIDs)
	if err != nil {
		return nil, fmt.Errorf("SolveRepo - GetSolvesForPointsRecalc: %w", err)
	}
	out := make([]*repo.SolveForPointsRecalc, 0, len(rows))
	for _, row := range rows {
		out = append(out, &repo.SolveForPointsRecalc{
			ID:           row.ID,
			ChallengeID:  row.ChallengeID,
			SolvedAt:     pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.SolvedAt)),
			InitialValue: int(row.InitialValue),
			MinValue:     int(row.MinValue),
			Decay:        int(row.Decay),
		})
	}
	return out, nil
}

func (r *SolveRepo) BatchUpdateSolvePoints(ctx context.Context, solveIDs []uuid.UUID, points []int) error {
	if len(solveIDs) == 0 {
		return nil
	}
	if len(solveIDs) != len(points) {
		return fmt.Errorf("SolveRepo - BatchUpdateSolvePoints: ids and points length mismatch (%d != %d)", len(solveIDs), len(points))
	}
	pts := make([]int32, len(points))
	for i, p := range points {
		pts[i] = int32(p)
	}
	if err := r.Q(ctx).BatchUpdateSolvePoints(ctx, sqlc.BatchUpdateSolvePointsParams{
		Column1: solveIDs,
		Column2: pts,
	}); err != nil {
		return fmt.Errorf("SolveRepo - BatchUpdateSolvePoints: %w", err)
	}
	return nil
}
