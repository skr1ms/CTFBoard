package competition

import (
	"context"
	"errors"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/scoring"
)

// RecordSolveInTx creates a solve and updates challenge solve count and points within the current transaction.
// Caller must hold a lock on the team row for solve.TeamID (e.g. via TeamRepo.Lock) before calling,
// so that concurrent submissions for the same team and challenge are serialized and IncrementSolveCount
// does not race.
func RecordSolveInTx(ctx context.Context, solve *domain.Solve, challenge *domain.Challenge, challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository) (solveCount int, err error) {
	_, err = solveRepo.GetByTeamAndChallengeForUpdate(ctx, solve.TeamID, solve.ChallengeID)
	if err == nil {
		return 0, httperr.ErrAlreadySolved
	}
	if !errors.Is(err, httperr.ErrSolveNotFound) {
		return 0, fmt.Errorf("RecordSolveInTx - GetByTeamAndChallengeForUpdate: %w", err)
	}
	solveCount, err = challengeRepo.IncrementSolveCount(ctx, solve.ChallengeID)
	if err != nil {
		return 0, fmt.Errorf("RecordSolveInTx - IncrementSolveCount: %w", err)
	}
	pointsAtSolve, err := scoring.ApplySolveScore(ctx,
		challenge.InitialValue, challenge.MinValue, challenge.Decay, challenge.Points, solveCount,
		func(ctx context.Context, pts int) error {
			if err := challengeRepo.UpdatePoints(ctx, challenge.ID, pts); err != nil {
				return fmt.Errorf("RecordSolveInTx - UpdatePoints: %w", err)
			}
			challenge.Points = pts
			return nil
		},
	)
	if err != nil {
		return 0, fmt.Errorf("RecordSolveInTx - ApplySolveScore: %w", err)
	}
	solve.PointsAtSolve = pointsAtSolve
	if err := solveRepo.Create(ctx, solve); err != nil {
		return 0, fmt.Errorf("RecordSolveInTx - SolveRepo.Create: %w", err)
	}
	return solveCount, nil
}
