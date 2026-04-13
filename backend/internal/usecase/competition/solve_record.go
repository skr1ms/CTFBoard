package competition

import (
	"context"
	"errors"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
)

// RecordSolveInTx creates a solve record and increments the challenge solve
// count inside the caller's transaction. The caller must hold a row-level lock
// on the team (e.g. via TeamRepo.Lock) before calling, so that concurrent
// submissions for the same team and challenge are serialized and
// IncrementSolveCount does not race
//
// After incrementing the solve count it applies dynamic score decay: the
// optional decayFn parameter selects the decay algorithm (defaults to
// scoring.DecayLogarithmic). scoring.ApplySolveScore computes the new
// challenge point value and updates challenge.Points in the database, then
// recalculates points for all previous solvers. The points value at the moment
// of this solve is stored on the Solve record (PointsAtSolve) before it is
// persisted, so historical scoreboard queries remain accurate even after
// subsequent decays.
func RecordSolveInTx(ctx context.Context, solve *domain.Solve, challenge *domain.Challenge, challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository, decayFn ...scoring.DecayFunction) (solveCount int, err error) {
	_, err = solveRepo.GetByTeamAndChallengeForUpdate(ctx, solve.TeamID, solve.ChallengeID)
	if err == nil {
		return 0, apperr.ErrAlreadySolved
	}

	if !errors.Is(err, apperr.ErrSolveNotFound) {
		return 0, fmt.Errorf("RecordSolveInTx - GetByTeamAndChallengeForUpdate: %w", err)
	}

	solveCount, err = challengeRepo.IncrementSolveCount(ctx, solve.ChallengeID)
	if err != nil {
		return 0, fmt.Errorf("RecordSolveInTx - IncrementSolveCount: %w", err)
	}

	fn := scoring.DecayLogarithmic

	if len(decayFn) > 0 && decayFn[0] != "" {
		fn = decayFn[0]
	}

	pointsAtSolve, err := scoring.ApplySolveScore(ctx,
		challenge.InitialValue, challenge.MinValue, challenge.Decay, challenge.Points, solveCount,
		fn,
		func(ctx context.Context, pts int) error {
			err := challengeRepo.UpdatePoints(ctx, challenge.ID, pts)
			if err != nil {
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
