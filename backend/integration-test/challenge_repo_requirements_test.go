package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallengeRepo_GetRequirements_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	mainCh := f.CreateChallenge(t, "main_req", 100)
	prereqCh := f.CreateChallenge(t, "prereq_req", 50)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.SetRequirements(txCtx, mainCh.ID, []uuid.UUID{prereqCh.ID})
	})
	require.NoError(t, err)

	got, err := f.ChallengeRepo.GetRequirements(ctx, mainCh.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, prereqCh.ID, got[0].ChallengeID)
	assert.Equal(t, prereqCh.Title, got[0].ChallengeTitle)
	assert.NotNil(t, got[0].Category)
	assert.Equal(t, "Web", *got[0].Category)
}

func TestChallengeRepo_GetRequirements_Empty(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "no_reqs", 100)

	got, err := f.ChallengeRepo.GetRequirements(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestChallengeRepo_SetRequirements_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	mainCh := f.CreateChallenge(t, "main_set", 200)
	prereq1 := f.CreateChallenge(t, "prereq1_set", 50)
	prereq2 := f.CreateChallenge(t, "prereq2_set", 75)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.SetRequirements(txCtx, mainCh.ID, []uuid.UUID{prereq1.ID, prereq2.ID})
	})
	require.NoError(t, err)

	got, err := f.ChallengeRepo.GetRequirements(ctx, mainCh.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	ids := map[uuid.UUID]bool{got[0].ChallengeID: true, got[1].ChallengeID: true}
	assert.True(t, ids[prereq1.ID])
	assert.True(t, ids[prereq2.ID])
}

func TestChallengeRepo_SetRequirements_InvalidRequiredID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	mainCh := f.CreateChallenge(t, "main_invalid", 100)
	nonExistentID := uuid.New()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.SetRequirements(txCtx, mainCh.ID, []uuid.UUID{nonExistentID})
	})

	assert.Error(t, err)
}
