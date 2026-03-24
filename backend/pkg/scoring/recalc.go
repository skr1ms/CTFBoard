package scoring

import (
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// RecalculatePoints returns (challengeIDs, points) for challenges that use dynamic scoring,
// for use with ChallengeRepo.BatchUpdatePoints. Challenges with InitialValue <= 0 or Decay <= 0 are skipped.
func RecalculatePoints(challengesMap map[uuid.UUID]*domain.Challenge) (ids []uuid.UUID, points []int) {
	for id, c := range challengesMap {
		if c == nil || c.InitialValue <= 0 || c.Decay <= 0 {
			continue
		}
		ids = append(ids, id)
		points = append(points, CalculateDynamicScore(c.InitialValue, c.MinValue, c.Decay, c.SolveCount))
	}
	return ids, points
}

// SolveRowForPointsRecalc is a row from GetSolvesForPointsRecalc. Rows must be ordered by challenge_id, solved_at.
type SolveRowForPointsRecalc struct {
	ID           uuid.UUID
	ChallengeID  uuid.UUID
	InitialValue int
	MinValue     int
	Decay        int
}

// RecalculatePointsAtSolveRows returns (solveIDs, points) for each solve in rows. Rows are grouped by
// challenge (consecutive same challenge_id); within each group rank is 1..n by order. Used after
// ban/unban to update points_at_solve so scoreboard stays consistent with challenge points.
func RecalculatePointsAtSolveRows(rows []*SolveRowForPointsRecalc) (solveIDs []uuid.UUID, points []int) {
	if len(rows) == 0 {
		return nil, nil
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
		pts := CalculateDynamicScore(row.InitialValue, row.MinValue, row.Decay, rank)
		points = append(points, pts)
	}
	return solveIDs, points
}
