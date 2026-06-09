package integration_test

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestSolveRepo_GetScoreboard_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "score_1")
	u2, t2 := f.CreateUserWithTeam(t, "score_2")

	ch1 := f.CreateChallenge(t, "score_ch1", 100)
	ch2 := f.CreateChallenge(t, "score_ch2", 200)

	f.CreateSolve(t, u1.ID, t1.ID, ch1.ID)
	f.CreateSolve(t, u1.ID, t1.ID, ch2.ID)
	f.CreateSolve(t, u2.ID, t2.ID, ch1.ID)

	require.Eventually(t, func() bool {
		sb, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
		if err != nil {
			return false
		}

		var t1Found, t2Found bool

		for _, e := range sb {
			if e.TeamID == t1.ID {
				if e.Points != 300 {
					return false
				}

				t1Found = true
			}

			if e.TeamID == t2.ID {
				if e.Points != 100 {
					return false
				}

				t2Found = true
			}
		}

		return t1Found && t2Found
	}, 2*time.Second, 20*time.Millisecond)

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	require.NoError(t, err)

	idx1 := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == t1.ID })
	require.GreaterOrEqual(t, idx1, 0, "t1 should be in scoreboard with 300 points")
	assert.Equal(t, t1.Name, scoreboard[idx1].TeamName)
	assert.Equal(t, 300, scoreboard[idx1].Points)
	idx2 := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == t2.ID })
	require.GreaterOrEqual(t, idx2, 0, "t2 should be in scoreboard with 100 points")
	assert.Equal(t, t2.Name, scoreboard[idx2].TeamName)
	assert.Equal(t, 100, scoreboard[idx2].Points)
}

func TestSolveRepo_GetScoreboard_Empty(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "empty_score")

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	require.NoError(t, err)

	idx := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == team.ID })
	require.GreaterOrEqual(t, idx, 0, "our team should appear in scoreboard with 0 points")
	assert.Equal(t, team.Name, scoreboard[idx].TeamName)
	assert.Equal(t, 0, scoreboard[idx].Points)
}

func TestSolveRepo_GetScoreboard_HiddenTeamNotIncluded(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "visible_team")
	u2, t2 := f.CreateUserWithTeam(t, "hidden_team")

	ch := f.CreateChallenge(t, "score_ch", 100)
	f.CreateSolve(t, u1.ID, t1.ID, ch.ID)
	f.CreateSolve(t, u2.ID, t2.ID, ch.ID)

	err := f.TeamRepo.SetHidden(ctx, t2.ID, true)
	require.NoError(t, err)

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	require.NoError(t, err)
	assert.True(t, slices.ContainsFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == t1.ID }), "visible team t1 should be in scoreboard")
	assert.False(t, slices.ContainsFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == t2.ID }), "hidden team t2 should not be in scoreboard")
}

func TestSolveRepo_GetScoreboard_BannedTeamNotIncluded(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "active_team")
	u2, t2 := f.CreateUserWithTeam(t, "banned_team")

	ch := f.CreateChallenge(t, "ban_score_ch", 100)
	f.CreateSolve(t, u1.ID, t1.ID, ch.ID)
	f.CreateSolve(t, u2.ID, t2.ID, ch.ID)

	err := f.TeamRepo.Ban(ctx, t2.ID, "test ban")
	require.NoError(t, err)

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	require.NoError(t, err)
	assert.True(t, slices.ContainsFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == t1.ID }), "active team t1 should be in scoreboard")
	assert.False(t, slices.ContainsFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == t2.ID }), "banned team t2 should not be in scoreboard")
}

func TestSolveRepo_GetScoreboard_BracketScope(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	bracketA := f.CreateBracket(t, "score_scope_a")
	bracketB := f.CreateBracket(t, "score_scope_b")

	uA, teamA := f.CreateUserWithTeam(t, "scope_a")
	uB, teamB := f.CreateUserWithTeam(t, "scope_b")
	_, teamGlobal := f.CreateUserWithTeam(t, "scope_global")
	challenge := f.CreateChallenge(t, "scope_challenge", 100)

	require.NoError(t, f.TeamRepo.SetBracket(ctx, teamA.ID, &bracketA.ID))
	require.NoError(t, f.TeamRepo.SetBracket(ctx, teamB.ID, &bracketB.ID))
	f.CreateSolve(t, uA.ID, teamA.ID, challenge.ID)
	f.CreateSolve(t, uB.ID, teamB.ID, challenge.ID)
	f.CreateAward(t, teamA.ID, 25, "scope award a", nil)
	f.CreateAward(t, teamB.ID, 50, "scope award b", nil)
	f.CreateAward(t, teamGlobal.ID, 75, "scope award global", nil)

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, &bracketA.ID, nil)
	require.NoError(t, err)

	assert.True(t, slices.ContainsFunc(scoreboard, func(e *domain.ScoreboardEntry) bool {
		return e.TeamID == teamA.ID && e.Points == 125
	}), "bracket A team should be included with solve and award points")
	assert.False(t, slices.ContainsFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == teamB.ID }), "bracket B team should not be included")
	assert.False(t, slices.ContainsFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == teamGlobal.ID }), "unbracketed team should not be included")
}

func TestSolveRepo_GetScoreboard_ExcludesSoftBannedSources(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	uSolveBanned, teamSolveBanned := f.CreateUserWithTeam(t, "source_solve_banned")
	_, teamAwardBanned := f.CreateUserWithTeam(t, "source_award_banned")
	challenge := f.CreateChallenge(t, "source_filter", 100)

	f.CreateSolve(t, uSolveBanned.ID, teamSolveBanned.ID, challenge.ID)
	f.CreateAward(t, teamSolveBanned.ID, 25, "award still active", nil)
	f.CreateAward(t, teamAwardBanned.ID, 75, "award to ban", nil)

	require.NoError(t, f.SolveRepo.SoftBanByTeamID(ctx, teamSolveBanned.ID))
	require.NoError(t, f.AwardRepo.SoftBanByTeamID(ctx, teamAwardBanned.ID))

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	require.NoError(t, err)

	solveBannedIdx := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == teamSolveBanned.ID })
	awardBannedIdx := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == teamAwardBanned.ID })

	require.GreaterOrEqual(t, solveBannedIdx, 0)
	require.GreaterOrEqual(t, awardBannedIdx, 0)
	assert.Equal(t, 25, scoreboard[solveBannedIdx].Points, "soft-banned solve points should be excluded while active awards still count")
	assert.Equal(t, 0, scoreboard[awardBannedIdx].Points, "soft-banned award points should be excluded while the team row remains visible")
}

func TestSolveRepo_GetFirstBlood(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "fb_1")
	u2, t2 := f.CreateUserWithTeam(t, "fb_2")
	ch := f.CreateChallenge(t, "fb_ch", 100)

	f.CreateSolve(t, u1.ID, t1.ID, ch.ID)
	f.CreateSolve(t, u2.ID, t2.ID, ch.ID)

	require.Eventually(t, func() bool {
		fb, err := f.SolveRepo.GetFirstBlood(ctx, ch.ID, nil)

		return err == nil && fb != nil && fb.UserID == u1.ID
	}, 2*time.Second, 50*time.Millisecond)

	firstBlood, err := f.SolveRepo.GetFirstBlood(ctx, ch.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, u1.ID, firstBlood.UserID)
	assert.Equal(t, u1.Username, firstBlood.Username)
	assert.Equal(t, t1.ID, firstBlood.TeamID)
	assert.Equal(t, t1.Name, firstBlood.TeamName)
}

func TestSolveRepo_GetFirstBlood_NoSolves(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch := f.CreateChallenge(t, "no_solves_ch", 100)

	_, err := f.SolveRepo.GetFirstBlood(ctx, ch.ID, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrSolveNotFound)
}

func TestSolveRepo_GetScoreboardFrozen(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "frozen")
	ch1 := f.CreateChallenge(t, "frozen_1", 100)
	ch2 := f.CreateChallenge(t, "frozen_2", 200)

	solve1 := f.CreateSolve(t, u1.ID, t1.ID, ch1.ID)

	backdated := time.Now().Add(-1 * time.Hour)
	_, err := f.Pool.Exec(ctx, "UPDATE solves SET solved_at = $1 WHERE ID = $2", backdated, solve1.ID)
	require.NoError(t, err)

	freezeTime := time.Now().Add(-30 * time.Minute)

	f.CreateSolve(t, u1.ID, t1.ID, ch2.ID)

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, &freezeTime)
	require.NoError(t, err)

	found := false

	for _, entry := range scoreboard {
		if entry.TeamID == t1.ID {
			assert.Equal(t, 100, entry.Points)

			found = true
		}
	}

	assert.True(t, found)
}

func TestSolveRepo_GetScoreboard_TieBreaksByLastSolveTime(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	uEarly, tEarly := f.CreateUserWithTeam(t, "tie_early")
	uLate, tLate := f.CreateUserWithTeam(t, "tie_late")
	challenge := f.CreateChallenge(t, "tie_break", 100)

	earlySolve := f.CreateSolve(t, uEarly.ID, tEarly.ID, challenge.ID)
	lateSolve := f.CreateSolve(t, uLate.ID, tLate.ID, challenge.ID)

	base := time.Now().UTC().Add(-2 * time.Hour)
	_, err := f.Pool.Exec(ctx, "UPDATE solves SET solved_at = $1 WHERE id = $2", base, earlySolve.ID)
	require.NoError(t, err)
	_, err = f.Pool.Exec(ctx, "UPDATE solves SET solved_at = $1 WHERE id = $2", base.Add(time.Minute), lateSolve.ID)
	require.NoError(t, err)

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	require.NoError(t, err)

	earlyIdx := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == tEarly.ID })
	lateIdx := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == tLate.ID })

	require.GreaterOrEqual(t, earlyIdx, 0)
	require.GreaterOrEqual(t, lateIdx, 0)
	require.Equal(t, 100, scoreboard[earlyIdx].Points)
	require.Equal(t, 100, scoreboard[lateIdx].Points)
	assert.Less(t, earlyIdx, lateIdx, "earlier last solve should rank higher at equal score")
}

func TestSolveRepo_GetScoreboard_ZeroPointSolveRanksBeforeNeverSolved(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	uSolved, tSolved := f.CreateUserWithTeam(t, "zero_solved")
	_, tNeverSolved := f.CreateUserWithTeam(t, "zero_never")
	challenge := f.CreateChallenge(t, "zero_point", 0)

	f.CreateSolve(t, uSolved.ID, tSolved.ID, challenge.ID)

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	require.NoError(t, err)

	solvedIdx := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == tSolved.ID })
	neverIdx := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == tNeverSolved.ID })

	require.GreaterOrEqual(t, solvedIdx, 0)
	require.GreaterOrEqual(t, neverIdx, 0)
	require.Equal(t, 0, scoreboard[solvedIdx].Points)
	require.Equal(t, 0, scoreboard[neverIdx].Points)
	assert.Less(t, solvedIdx, neverIdx, "a counted solve should rank before a never-solved team at equal score")
}

func TestSolveRepo_GetScoreboard_AwardsRespectFreezeCutoff(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, team := f.CreateUserWithTeam(t, "award_freeze")
	challenge := f.CreateChallenge(t, "award_freeze", 100)
	solve := f.CreateSolve(t, u1.ID, team.ID, challenge.ID)
	beforeAward := f.CreateAward(t, team.ID, 50, "before freeze", nil)
	afterAward := f.CreateAward(t, team.ID, 75, "after freeze", nil)

	base := time.Now().UTC().Add(-3 * time.Hour)
	freezeTime := base.Add(time.Hour)

	_, err := f.Pool.Exec(ctx, "UPDATE solves SET solved_at = $1 WHERE id = $2", base, solve.ID)
	require.NoError(t, err)
	_, err = f.Pool.Exec(ctx, "UPDATE awards SET created_at = $1 WHERE id = $2", base.Add(30*time.Minute), beforeAward.ID)
	require.NoError(t, err)
	_, err = f.Pool.Exec(ctx, "UPDATE awards SET created_at = $1 WHERE id = $2", freezeTime.Add(30*time.Minute), afterAward.ID)
	require.NoError(t, err)

	frozen, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, &freezeTime)
	require.NoError(t, err)
	live, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	require.NoError(t, err)

	frozenIdx := slices.IndexFunc(frozen, func(e *domain.ScoreboardEntry) bool { return e.TeamID == team.ID })
	liveIdx := slices.IndexFunc(live, func(e *domain.ScoreboardEntry) bool { return e.TeamID == team.ID })

	require.GreaterOrEqual(t, frozenIdx, 0)
	require.GreaterOrEqual(t, liveIdx, 0)
	assert.Equal(t, 150, frozen[frozenIdx].Points)
	assert.Equal(t, 225, live[liveIdx].Points)
}

func TestSolveRepo_GetScoreboard_AwardOnlyTieFallsBackToTeamID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, t1 := f.CreateUserWithTeam(t, "award_tie_1")
	_, t2 := f.CreateUserWithTeam(t, "award_tie_2")
	a1 := f.CreateAward(t, t1.ID, 100, "award-only tie 1", nil)
	a2 := f.CreateAward(t, t2.ID, 100, "award-only tie 2", nil)

	base := time.Now().UTC().Add(-2 * time.Hour)
	_, err := f.Pool.Exec(ctx, "UPDATE awards SET created_at = $1 WHERE id = $2", base.Add(time.Hour), a1.ID)
	require.NoError(t, err)
	_, err = f.Pool.Exec(ctx, "UPDATE awards SET created_at = $1 WHERE id = $2", base, a2.ID)
	require.NoError(t, err)

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	require.NoError(t, err)

	t1Idx := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == t1.ID })
	t2Idx := slices.IndexFunc(scoreboard, func(e *domain.ScoreboardEntry) bool { return e.TeamID == t2.ID })

	require.GreaterOrEqual(t, t1Idx, 0)
	require.GreaterOrEqual(t, t2Idx, 0)
	require.Equal(t, 100, scoreboard[t1Idx].Points)
	require.Equal(t, 100, scoreboard[t2Idx].Points)

	firstIdx, secondIdx := t1Idx, t2Idx

	if bytes.Compare(t1.ID[:], t2.ID[:]) > 0 {
		firstIdx, secondIdx = t2Idx, t1Idx
	}

	assert.Less(t, firstIdx, secondIdx, "award-only ties should ignore award time and fall back to team ID")
}
