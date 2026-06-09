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

func TestSolveRepo_Create_AllowsActiveReplacementAfterUserSoftBan(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	firstUser, team := f.CreateUserWithTeam(t, "soft_re_solve_1")
	secondUser := f.CreateUser(t, "soft_re_solve_2")
	f.AddUserToTeam(t, secondUser.ID, team.ID)
	challenge := f.CreateChallenge(t, "soft_re_solve_ch", 100)

	firstSolve := f.CreateSolve(t, firstUser.ID, team.ID, challenge.ID)

	require.NoError(t, f.SolveRepo.SoftBanByTeamIDAndUserID(ctx, team.ID, firstUser.ID))
	secondSolve := f.CreateSolve(t, secondUser.ID, team.ID, challenge.ID)

	activeSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, secondSolve.ID, activeSolve.ID)
	assert.NotEqual(t, firstSolve.ID, activeSolve.ID)

	require.NoError(t, f.SolveRepo.RestoreByBannedUserID(ctx, firstUser.ID))

	activeSolve, err = f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, secondSolve.ID, activeSolve.ID)

	var activeCount int

	err = f.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM solves
		WHERE team_id = $1
		  AND challenge_id = $2
		  AND banned_team_id IS NULL
		  AND banned_user_id IS NULL
	`, team.ID, challenge.ID).Scan(&activeCount)
	require.NoError(t, err)
	assert.Equal(t, 1, activeCount)
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
