package scoring

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// RecalculatePoints returns (challengeIDs, points) for challenges that use dynamic scoring,
// for use with ChallengeRepo.BatchUpdatePoints. Challenges with InitialValue <= 0 or Decay <= 0 are skipped.
func RecalculatePoints(challengesMap map[uuid.UUID]*domain.Challenge, fn ...DecayFunction) (ids []uuid.UUID, points []int) {
	decayFn := DecayLogarithmic

	if len(fn) > 0 && fn[0] != "" {
		decayFn = fn[0]
	}

	for id, c := range challengesMap {
		if c == nil || c.InitialValue <= 0 || c.Decay <= 0 {
			continue
		}

		ids = append(ids, id)
		points = append(points, CalculateScore(decayFn, c.InitialValue, c.MinValue, c.Decay, c.SolveCount))
	}

	return ids, points
}

// SolveRowForPointsRecalc is a row from GetSolvesForPointsRecalc. Rows must be ordered by challenge_id, solved_at.
type SolveRowForPointsRecalc struct {
	ID           uuid.UUID
	ChallengeID  uuid.UUID
	SolvedAt     time.Time
	InitialValue int
	MinValue     int
	Decay        int
}

// RecalculatePointsAtSolveRows returns (solveIDs, points) for each solve in rows. Rows are grouped by
// challenge (consecutive same challenge_id); within each group rank is 1..n by order. Used after
// ban/unban to update points_at_solve so scoreboard stays consistent with challenge points.
func RecalculatePointsAtSolveRows(rows []*SolveRowForPointsRecalc, fn ...DecayFunction) (solveIDs []uuid.UUID, points []int) {
	if len(rows) == 0 {
		return nil, nil
	}

	decayFn := DecayLogarithmic

	if len(fn) > 0 && fn[0] != "" {
		decayFn = fn[0]
	}

	solveIDs = make([]uuid.UUID, 0, len(rows))
	points = make([]int, 0, len(rows))

	var prevChallengeID uuid.UUID

	rank := 0

	for _, row := range rows {
		if row.ChallengeID != prevChallengeID {
			prevChallengeID = row.ChallengeID
			rank = 0
		}

		rank++

		solveIDs = append(solveIDs, row.ID)
		pts := CalculateScore(decayFn, row.InitialValue, row.MinValue, row.Decay, rank)
		points = append(points, pts)
	}

	return solveIDs, points
}

// CompetitionGetter is the minimal interface for retrieving the current competition.
type CompetitionGetter interface {
	Get(ctx context.Context) (*domain.Competition, error)
}

// FilterSolveDetailsByFreezeFromRepo loads the competition via compRepo and filters
// solves using FilterSolvesByFreeze. Returns solves unchanged when compRepo is nil.
func FilterSolveDetailsByFreezeFromRepo(ctx context.Context, compRepo CompetitionGetter, solves []*domain.SolveWithDetails) ([]*domain.SolveWithDetails, error) {
	if compRepo == nil {
		return solves, nil
	}

	comp, err := compRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("FilterSolveDetailsByFreezeFromRepo - Get: %w", err)
	}

	return FilterSolvesByFreeze(comp, solves), nil
}

// FilterSolvesByFreeze removes solves that happened after the competition freeze time.
// If comp is nil or freeze is not active, returns solves unchanged.
func FilterSolvesByFreeze(comp *domain.Competition, solves []*domain.SolveWithDetails) []*domain.SolveWithDetails {
	if comp == nil || !comp.IsFreezeActive() || comp.FreezeTime == nil {
		return solves
	}

	freezeAt := *comp.FreezeTime
	filtered := make([]*domain.SolveWithDetails, 0, len(solves))

	for _, s := range solves {
		if !s.SolvedAt.After(freezeAt) {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

// MapSolvesForRecalcFn builds a closure that fetches rows via getRows, maps each row to
// SolveRowForPointsRecalc using toRow, and returns the resulting slice.
// callerLabel is included in error messages (e.g. "UserUseCase - mapSolvesForRecalc").
func MapSolvesForRecalcFn[R any](
	getRows func(ctx context.Context, ids []uuid.UUID) ([]*R, error),
	toRow func(*R) *SolveRowForPointsRecalc,
	callerLabel string,
) func(ctx context.Context, ids []uuid.UUID) ([]*SolveRowForPointsRecalc, error) {
	if getRows == nil || toRow == nil {
		return nil
	}

	return func(ctx context.Context, ids []uuid.UUID) ([]*SolveRowForPointsRecalc, error) {
		rows, err := getRows(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("%s - GetSolvesForPointsRecalc: %w", callerLabel, err)
		}

		out := make([]*SolveRowForPointsRecalc, len(rows))
		for i, r := range rows {
			out[i] = toRow(r)
		}

		return out, nil
	}
}

// ChallengeScoreUpdater is the minimal interface for challenge score recalculation.
type ChallengeScoreUpdater interface {
	RecalculateSolveCounts(ctx context.Context, ids []uuid.UUID) error
	GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Challenge, error)
	BatchUpdatePoints(ctx context.Context, ids []uuid.UUID, points []int) error
}

// AdjustDynamicScores recalculates solve counts, challenge points, and per-solve
// point snapshots for the given challenge IDs. This is the single entry point for the
// 3-step recalc pattern used after ban/unban, team switch, and admin solve delete.
//
// getSolvesForRecalc and batchUpdateSolvePoints are optional (nil = skip per-solve recalc).
func AdjustDynamicScores(
	ctx context.Context,
	challengeIDs []uuid.UUID,
	challenges ChallengeScoreUpdater,
	getSolvesForRecalc func(ctx context.Context, ids []uuid.UUID) ([]*SolveRowForPointsRecalc, error),
	batchUpdateSolvePoints func(ctx context.Context, ids []uuid.UUID, points []int) error,
	fn DecayFunction,
) error {
	if len(challengeIDs) == 0 {
		return nil
	}

	if err := challenges.RecalculateSolveCounts(ctx, challengeIDs); err != nil {
		return fmt.Errorf("AdjustDynamicScores - RecalculateSolveCounts: %w", err)
	}

	challengesMap, err := challenges.GetByIDs(ctx, challengeIDs)
	if err != nil {
		return fmt.Errorf("AdjustDynamicScores - GetByIDs: %w", err)
	}

	ids, points := RecalculatePoints(challengesMap, fn)
	if len(ids) > 0 {
		if err := challenges.BatchUpdatePoints(ctx, ids, points); err != nil {
			return fmt.Errorf("AdjustDynamicScores - BatchUpdatePoints: %w", err)
		}
	}

	if getSolvesForRecalc == nil || batchUpdateSolvePoints == nil {
		return nil
	}

	var dynamicIDs []uuid.UUID

	for _, id := range challengeIDs {
		if c := challengesMap[id]; c != nil && c.InitialValue > 0 && c.Decay > 0 {
			dynamicIDs = append(dynamicIDs, id)
		}
	}

	if len(dynamicIDs) == 0 {
		return nil
	}

	rows, err := getSolvesForRecalc(ctx, dynamicIDs)
	if err != nil {
		return fmt.Errorf("AdjustDynamicScores - GetSolvesForPointsRecalc: %w", err)
	}

	solveIDs, newPoints := RecalculatePointsAtSolveRows(rows, fn)
	if len(solveIDs) > 0 {
		if err := batchUpdateSolvePoints(ctx, solveIDs, newPoints); err != nil {
			return fmt.Errorf("AdjustDynamicScores - BatchUpdateSolvePoints: %w", err)
		}
	}

	return nil
}

// SolveForPointsRecalc is a minimal solve projection used by SolveRepository.GetSolvesForPointsRecalc.
type SolveForPointsRecalc struct {
	ID           uuid.UUID
	ChallengeID  uuid.UUID
	SolvedAt     time.Time
	InitialValue int
	MinValue     int
	Decay        int
}

// DefaultSolveMapper converts a SolveForPointsRecalc row to SolveRowForPointsRecalc.
// Use with MapSolvesForRecalcFn to avoid duplicating the same inline lambda across callers.
func DefaultSolveMapper(r *SolveForPointsRecalc) *SolveRowForPointsRecalc {
	return &SolveRowForPointsRecalc{
		ID:           r.ID,
		ChallengeID:  r.ChallengeID,
		SolvedAt:     r.SolvedAt,
		InitialValue: r.InitialValue,
		MinValue:     r.MinValue,
		Decay:        r.Decay,
	}
}
