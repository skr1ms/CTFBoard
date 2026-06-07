package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
