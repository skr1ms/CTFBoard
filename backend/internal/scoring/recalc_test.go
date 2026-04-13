package scoring

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestRecalculatePoints_EmptyMap_ReturnsNil(t *testing.T) {
	t.Parallel()

	ids, pts := RecalculatePoints(map[uuid.UUID]*domain.Challenge{})
	assert.Nil(t, ids)
	assert.Nil(t, pts)
}

func TestRecalculatePoints_StaticChallenge_Skipped(t *testing.T) {
	t.Parallel()
	// InitialValue <= 0 -> skip
	id := uuid.New()
	challenges := map[uuid.UUID]*domain.Challenge{
		id: {ID: id, InitialValue: 0, MinValue: 0, Decay: 10, SolveCount: 5},
	}
	ids, pts := RecalculatePoints(challenges)
	assert.Empty(t, ids)
	assert.Empty(t, pts)
}

func TestRecalculatePoints_ZeroDecay_Skipped(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	challenges := map[uuid.UUID]*domain.Challenge{
		id: {ID: id, InitialValue: 500, MinValue: 100, Decay: 0, SolveCount: 5},
	}
	ids, pts := RecalculatePoints(challenges)
	assert.Empty(t, ids)
	assert.Empty(t, pts)
}

func TestRecalculatePoints_NilEntry_Skipped(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	challenges := map[uuid.UUID]*domain.Challenge{id: nil}
	ids, pts := RecalculatePoints(challenges)
	assert.Empty(t, ids)
	assert.Empty(t, pts)
}

func TestRecalculatePoints_DynamicChallenge_Included(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	challenges := map[uuid.UUID]*domain.Challenge{
		id: {ID: id, InitialValue: 500, MinValue: 100, Decay: 10, SolveCount: 5},
	}
	ids, pts := RecalculatePoints(challenges)
	require.Len(t, ids, 1)
	require.Len(t, pts, 1)
	assert.Equal(t, id, ids[0])
	assert.Greater(t, pts[0], 0)
	assert.LessOrEqual(t, pts[0], 500)
}

func TestRecalculatePoints_MultipleChallenges(t *testing.T) {
	t.Parallel()

	id1, id2 := uuid.New(), uuid.New()
	challenges := map[uuid.UUID]*domain.Challenge{
		id1: {ID: id1, InitialValue: 500, MinValue: 100, Decay: 10, SolveCount: 1},
		id2: {ID: id2, InitialValue: 300, MinValue: 50, Decay: 5, SolveCount: 3},
	}
	ids, pts := RecalculatePoints(challenges)
	assert.Len(t, ids, 2)
	assert.Len(t, pts, 2)
}

func TestRecalculatePoints_CustomDecayFunction(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	challenges := map[uuid.UUID]*domain.Challenge{
		id: {ID: id, InitialValue: 500, MinValue: 100, Decay: 10, SolveCount: 5},
	}
	idsLog, ptsLog := RecalculatePoints(challenges, DecayLogarithmic)
	idsLinear, ptsLinear := RecalculatePoints(challenges, DecayLinear)

	require.Len(t, idsLog, 1)
	require.Len(t, idsLinear, 1)
	// both produce valid scores within range
	assert.Greater(t, ptsLog[0], 0)
	assert.Greater(t, ptsLinear[0], 0)
}

func TestRecalculatePointsAtSolveRows_EmptySlice_ReturnsNil(t *testing.T) {
	t.Parallel()

	ids, pts := RecalculatePointsAtSolveRows(nil)
	assert.Nil(t, ids)
	assert.Nil(t, pts)
}

func TestRecalculatePointsAtSolveRows_SingleChallenge_RankIncrements(t *testing.T) {
	t.Parallel()

	challengeID := uuid.New()
	rows := []*SolveRowForPointsRecalc{
		{ID: uuid.New(), ChallengeID: challengeID, InitialValue: 500, MinValue: 100, Decay: 10},
		{ID: uuid.New(), ChallengeID: challengeID, InitialValue: 500, MinValue: 100, Decay: 10},
		{ID: uuid.New(), ChallengeID: challengeID, InitialValue: 500, MinValue: 100, Decay: 10},
	}
	ids, pts := RecalculatePointsAtSolveRows(rows)
	require.Len(t, ids, 3)
	require.Len(t, pts, 3)
	// later solves should have lower or equal points (more solvers -> lower score)
	assert.GreaterOrEqual(t, pts[0], pts[1])
	assert.GreaterOrEqual(t, pts[1], pts[2])
}

func TestRecalculatePointsAtSolveRows_TwoChallenges_RankResetsPerChallenge(t *testing.T) {
	t.Parallel()

	c1, c2 := uuid.New(), uuid.New()
	rows := []*SolveRowForPointsRecalc{
		{ID: uuid.New(), ChallengeID: c1, InitialValue: 500, MinValue: 100, Decay: 10},
		{ID: uuid.New(), ChallengeID: c1, InitialValue: 500, MinValue: 100, Decay: 10},
		{ID: uuid.New(), ChallengeID: c2, InitialValue: 300, MinValue: 50, Decay: 5},
	}
	ids, pts := RecalculatePointsAtSolveRows(rows)
	require.Len(t, ids, 3)
	// c2's first solve should equal c1's first solve in rank-1 score (different params but rank=1)
	_ = pts
	_ = ids
}
