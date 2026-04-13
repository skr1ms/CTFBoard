package integration_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

func TestSolveRepo_Create(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "solve")
	challenge := f.CreateChallenge(t, "solve_ch", 100)

	solve := f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, solve.TeamID, solve.ChallengeID)
	require.NoError(t, err)
	assert.NotEmpty(t, gotSolve.ID)
	assert.False(t, gotSolve.SolvedAt.IsZero())
}

func TestSolveRepo_Create_Duplicate(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "duplicate")
	challenge := f.CreateChallenge(t, "duplicate_ch", 100)

	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	solve2 := &domain.Solve{
		UserID:      user.ID,
		TeamID:      team.ID,
		ChallengeID: challenge.ID,
	}
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.SolveRepo.Create(txCtx, solve2)
	})
	assert.Error(t, err)
}

func TestSolveRepo_GetByID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_by_ID")
	challenge := f.CreateChallenge(t, "get_by_ID_ch", 100)
	solve := f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	gotSolve, err := f.SolveRepo.GetByID(ctx, solve.ID)
	require.NoError(t, err)
	assert.Equal(t, solve.ID, gotSolve.ID)
	assert.Equal(t, solve.UserID, gotSolve.UserID)
	assert.Equal(t, solve.TeamID, gotSolve.TeamID)
	assert.Equal(t, solve.ChallengeID, gotSolve.ChallengeID)
}

func TestSolveRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	nonExistentID := uuid.New()
	_, err := f.SolveRepo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrSolveNotFound)
}

func TestSolveRepo_GetByTeamAndChallenge(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_by_team")
	challenge := f.CreateChallenge(t, "get_by_team_ch", 100)
	solve := f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, solve.ID, gotSolve.ID)
	assert.Equal(t, team.ID, gotSolve.TeamID)
	assert.Equal(t, challenge.ID, gotSolve.ChallengeID)
}

func TestSolveRepo_GetByTeamAndChallenge_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "not_found")
	challenge := f.CreateChallenge(t, "not_found_ch", 100)

	_, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrSolveNotFound)
}

func TestSolveRepo_GetSolvedChallengeIDsByTeam_EmptyChallengeIDs(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "batch_empty")

	ids, err := f.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, team.ID, nil)
	require.NoError(t, err)
	assert.Nil(t, ids)

	ids, err = f.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, team.ID, []uuid.UUID{})
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestSolveRepo_GetSolvedChallengeIDsByTeam_NoneSolved(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "batch_none")
	ch1 := f.CreateChallenge(t, "batch_ch1", 100)
	ch2 := f.CreateChallenge(t, "batch_ch2", 200)

	ids, err := f.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, team.ID, []uuid.UUID{ch1.ID, ch2.ID})
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestSolveRepo_GetSolvedChallengeIDsByTeam_SubsetSolved(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "batch_subset")
	ch1 := f.CreateChallenge(t, "batch_sub_ch1", 100)
	ch2 := f.CreateChallenge(t, "batch_sub_ch2", 200)
	ch3 := f.CreateChallenge(t, "batch_sub_ch3", 300)

	f.CreateSolve(t, user.ID, team.ID, ch1.ID)
	f.CreateSolve(t, user.ID, team.ID, ch3.ID)

	ids, err := f.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, team.ID, []uuid.UUID{ch1.ID, ch2.ID, ch3.ID})
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, ch1.ID)
	assert.Contains(t, ids, ch3.ID)
	assert.NotContains(t, ids, ch2.ID)
}

func TestSolveRepo_GetSolvedChallengeIDsByTeam_AllSolved(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "batch_all")
	ch1 := f.CreateChallenge(t, "batch_all_ch1", 100)
	ch2 := f.CreateChallenge(t, "batch_all_ch2", 200)

	f.CreateSolve(t, user.ID, team.ID, ch1.ID)
	f.CreateSolve(t, user.ID, team.ID, ch2.ID)

	ids, err := f.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, team.ID, []uuid.UUID{ch1.ID, ch2.ID})
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, ch1.ID)
	assert.Contains(t, ids, ch2.ID)
}

func TestSolveRepo_GetSolvedChallengeIDsByTeam_OnlyRequestedIdsReturned(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "batch_requested")
	ch1 := f.CreateChallenge(t, "batch_req_ch1", 100)
	ch2 := f.CreateChallenge(t, "batch_req_ch2", 200)

	f.CreateSolve(t, user.ID, team.ID, ch1.ID)
	f.CreateSolve(t, user.ID, team.ID, ch2.ID)

	ids, err := f.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, team.ID, []uuid.UUID{ch1.ID})
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	assert.Equal(t, ch1.ID, ids[0])
}

func TestSolveRepo_GetByUserID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_by_user")
	ch1 := f.CreateChallenge(t, "ch1", 100)
	ch2 := f.CreateChallenge(t, "ch2", 200)

	f.CreateSolve(t, user.ID, team.ID, ch1.ID)
	f.CreateSolve(t, user.ID, team.ID, ch2.ID)

	require.Eventually(t, func() bool {
		solves, err := f.SolveRepo.GetByUserID(ctx, user.ID)

		return err == nil && len(solves) == 2 && solves[0].ChallengeID == ch2.ID && solves[1].ChallengeID == ch1.ID
	}, 2*time.Second, 50*time.Millisecond)

	solves, err := f.SolveRepo.GetByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, solves, 2)
	assert.Equal(t, ch2.ID, solves[0].ChallengeID)
	assert.Equal(t, ch1.ID, solves[1].ChallengeID)
}

func TestSolveRepo_GetByUserID_Success_Empty(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "empty")

	solves, err := f.SolveRepo.GetByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Empty(t, solves)
}

func TestSolveRepo_GetByUserID_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "get_by_user_err")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	solves, err := f.SolveRepo.GetByUserID(ctx, user.ID)
	assert.Error(t, err)
	assert.Nil(t, solves)
}

func TestSolveRepo_GetAll_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "get_all_1")
	u2, t2 := f.CreateUserWithTeam(t, "get_all_2")
	ch1 := f.CreateChallenge(t, "get_all_ch1", 100)
	ch2 := f.CreateChallenge(t, "get_all_ch2", 200)

	solve1 := f.CreateSolve(t, u1.ID, t1.ID, ch1.ID)
	solve2 := f.CreateSolve(t, u2.ID, t2.ID, ch2.ID)

	solves, err := f.SolveRepo.GetAll(ctx)
	require.NoError(t, err)
	assert.True(t, slices.ContainsFunc(solves, func(s *domain.Solve) bool { return s.ID == solve1.ID }), "solve1 should be in GetAll result")
	assert.True(t, slices.ContainsFunc(solves, func(s *domain.Solve) bool { return s.ID == solve2.ID }), "solve2 should be in GetAll result")
}

func TestSolveRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	solves, err := f.SolveRepo.GetAll(ctx)
	assert.Error(t, err)
	assert.Nil(t, solves)
}

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

	idx1 := slices.IndexFunc(scoreboard, func(e *repo.ScoreboardEntry) bool { return e.TeamID == t1.ID })
	require.GreaterOrEqual(t, idx1, 0, "t1 should be in scoreboard with 300 points")
	assert.Equal(t, t1.Name, scoreboard[idx1].TeamName)
	assert.Equal(t, 300, scoreboard[idx1].Points)
	idx2 := slices.IndexFunc(scoreboard, func(e *repo.ScoreboardEntry) bool { return e.TeamID == t2.ID })
	require.GreaterOrEqual(t, idx2, 0, "t2 should be in scoreboard with 100 points")
	assert.Equal(t, t2.Name, scoreboard[idx2].TeamName)
	assert.Equal(t, 100, scoreboard[idx2].Points)
}

func TestSolveRepo_GetScoreboard_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, scoreboard)
}

func TestSolveRepo_GetScoreboard_Empty(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "empty_score")

	scoreboard, err := f.SolveRepo.GetScoreboardByBracket(ctx, nil, nil)
	require.NoError(t, err)

	idx := slices.IndexFunc(scoreboard, func(e *repo.ScoreboardEntry) bool { return e.TeamID == team.ID })
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
	assert.True(t, slices.ContainsFunc(scoreboard, func(e *repo.ScoreboardEntry) bool { return e.TeamID == t1.ID }), "visible team t1 should be in scoreboard")
	assert.False(t, slices.ContainsFunc(scoreboard, func(e *repo.ScoreboardEntry) bool { return e.TeamID == t2.ID }), "hidden team t2 should not be in scoreboard")
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
	assert.True(t, slices.ContainsFunc(scoreboard, func(e *repo.ScoreboardEntry) bool { return e.TeamID == t1.ID }), "active team t1 should be in scoreboard")
	assert.False(t, slices.ContainsFunc(scoreboard, func(e *repo.ScoreboardEntry) bool { return e.TeamID == t2.ID }), "banned team t2 should not be in scoreboard")
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

func TestSolveRepo_CreateTx(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "tx_create")
	ch := f.CreateChallenge(t, "tx_create_ch", 100)

	solve := &domain.Solve{
		UserID:      u.ID,
		TeamID:      tTeam.ID,
		ChallengeID: ch.ID,
	}
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.SolveRepo.Create(txCtx, solve)
	})
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, tTeam.ID, ch.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, gotSolve.ID)
	assert.Equal(t, u.ID, gotSolve.UserID)
}

func TestSolveRepo_CreateTx_Rollback(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "tx_rollback")
	ch := f.CreateChallenge(t, "tx_rollback_ch", 100)

	solve := &domain.Solve{
		UserID:      u.ID,
		TeamID:      tTeam.ID,
		ChallengeID: ch.ID,
	}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		err := f.SolveRepo.Create(txCtx, solve)
		if err != nil {
			return err
		}

		return errors.New("rollback")
	})
	assert.Error(t, err)

	_, err = f.SolveRepo.GetByTeamAndChallenge(ctx, tTeam.ID, ch.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrSolveNotFound)
}

func TestSolveRepo_GetByTeamAndChallengeTx(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "get_tx")
	ch := f.CreateChallenge(t, "get_tx_ch", 100)
	f.CreateSolve(t, u.ID, tTeam.ID, ch.ID)

	var gotSolve *domain.Solve

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		var err error

		gotSolve, err = f.SolveRepo.GetByTeamAndChallengeForUpdate(txCtx, tTeam.ID, ch.ID)

		return err
	})
	require.NoError(t, err)
	assert.Equal(t, tTeam.ID, gotSolve.TeamID)
	assert.Equal(t, ch.ID, gotSolve.ChallengeID)
}

func TestSolveRepo_GetByTeamAndChallengeTx_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, tTeam := f.CreateUserWithTeam(t, "not_found_tx")
	ch := f.CreateChallenge(t, "not_found_tx_ch", 100)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		_, err := f.SolveRepo.GetByTeamAndChallengeForUpdate(txCtx, tTeam.ID, ch.ID)

		return err
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrSolveNotFound)
}

func TestSolveRepo_GetTeamScoreTx(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "score_tx")
	ch1 := f.CreateChallenge(t, "score_tx_1", 100)
	ch2 := f.CreateChallenge(t, "score_tx_2", 200)

	f.CreateSolve(t, u.ID, tTeam.ID, ch1.ID)
	f.CreateSolve(t, u.ID, tTeam.ID, ch2.ID)

	var score int

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		var err error

		score, err = f.SolveRepo.GetTeamScore(txCtx, tTeam.ID)

		return err
	})
	require.NoError(t, err)
	assert.Equal(t, 300, score)
}

func TestSolveRepo_AtomicSubmitFlow(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "atomic")
	initialValue, minValue, decay := 500, 100, 1
	ch := f.CreateDynamicChallenge(t, "atomic_ch", initialValue, minValue, decay)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		_, err := f.SolveRepo.GetByTeamAndChallengeForUpdate(txCtx, tTeam.ID, ch.ID)
		if err == nil {
			return errors.New("expected not found")
		}

		if !errors.Is(err, apperr.ErrSolveNotFound) {
			return err
		}

		gotChallenge, err := f.ChallengeRepo.GetByIDForUpdate(txCtx, ch.ID)
		if err != nil {
			return err
		}

		solve := &domain.Solve{UserID: u.ID, TeamID: tTeam.ID, ChallengeID: ch.ID}
		if err := f.SolveRepo.Create(txCtx, solve); err != nil {
			return err
		}

		solveCount := gotChallenge.SolveCount + 1

		newPoints := max(int(float64(gotChallenge.MinValue)+(float64(gotChallenge.InitialValue-gotChallenge.MinValue)/(1+float64(solveCount-1)/float64(gotChallenge.Decay)))), gotChallenge.MinValue)

		_, err = f.ChallengeRepo.IncrementSolveCount(txCtx, ch.ID)
		if err != nil {
			return err
		}

		return f.ChallengeRepo.UpdatePoints(txCtx, ch.ID, newPoints)
	})
	require.NoError(t, err)

	finalChallenge, err := f.ChallengeRepo.GetByID(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, finalChallenge.SolveCount)
	assert.Equal(t, initialValue, finalChallenge.Points)

	finalSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, tTeam.ID, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, finalSolve.UserID)
}

func TestSolveRepo_SoftBanByTeamIDAndUserID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "softban")
	ch := f.CreateChallenge(t, "softban_ch", 100)
	solve := f.CreateSolve(t, user.ID, team.ID, ch.ID)

	err := f.SolveRepo.SoftBanByTeamIDAndUserID(ctx, team.ID, user.ID)
	require.NoError(t, err)

	// solve should now be soft-banned (banned_user_id set to user_id)
	row := f.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM solves WHERE id = $1 AND banned_user_id IS NOT NULL", solve.ID)

	var count int
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count, "solve should have banned_user_id set after SoftBan")
}

func TestSolveRepo_RestoreByBannedUserID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "restore")
	ch := f.CreateChallenge(t, "restore_ch", 100)
	solve := f.CreateSolve(t, user.ID, team.ID, ch.ID)

	// Soft-ban the solve first
	require.NoError(t, f.SolveRepo.SoftBanByTeamIDAndUserID(ctx, team.ID, user.ID))

	// Now restore
	err := f.SolveRepo.RestoreByBannedUserID(ctx, user.ID)
	require.NoError(t, err)

	row := f.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM solves WHERE id = $1 AND banned_user_id IS NULL", solve.ID)

	var count int
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count, "solve should have banned_user_id cleared after restore")
}

func TestSolveRepo_BatchUpdateSolvePoints(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user1, team1 := f.CreateUserWithTeam(t, "bsp1")
	user2, team2 := f.CreateUserWithTeam(t, "bsp2")
	ch := f.CreateChallenge(t, "bsp_ch", 100)

	solve1 := f.CreateSolve(t, user1.ID, team1.ID, ch.ID)
	solve2 := f.CreateSolve(t, user2.ID, team2.ID, ch.ID)

	err := f.SolveRepo.BatchUpdateSolvePoints(ctx, []uuid.UUID{solve1.ID, solve2.ID}, []int{150, 175})
	require.NoError(t, err)

	row1 := f.Pool.QueryRow(ctx, "SELECT points_at_solve FROM solves WHERE id = $1", solve1.ID)

	var pts1 int
	require.NoError(t, row1.Scan(&pts1))
	assert.Equal(t, 150, pts1)

	row2 := f.Pool.QueryRow(ctx, "SELECT points_at_solve FROM solves WHERE id = $1", solve2.ID)

	var pts2 int
	require.NoError(t, row2.Scan(&pts2))
	assert.Equal(t, 175, pts2)
}
