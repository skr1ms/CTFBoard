package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

func TestChallengeRepo_GetByIDTx(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "tx_get", 200, 100, 20)
	_, err := f.Pool.Exec(ctx, "UPDATE challenges SET solve_count = 5 WHERE ID = $1", challenge.ID)
	require.NoError(t, err)

	challenge.SolveCount = 5

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		gotChallenge, err := f.ChallengeRepo.GetByIDForUpdate(txCtx, challenge.ID)
		if err != nil {
			return err
		}

		_ = gotChallenge

		return nil
	})
	require.NoError(t, err)
	gotChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, challenge.ID, gotChallenge.ID)
	assert.Equal(t, challenge.Title, gotChallenge.Title)
	assert.Equal(t, challenge.Points, gotChallenge.Points)
	assert.Equal(t, challenge.SolveCount, gotChallenge.SolveCount)
}

func TestChallengeRepo_GetByIDTx_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	nonExistentID := uuid.New()
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		_, err := f.ChallengeRepo.GetByIDForUpdate(txCtx, nonExistentID)

		return err
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
}

func TestChallengeRepo_GetByIDs_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "getbyids_1", 100)
	ch2 := f.CreateChallenge(t, "getbyids_2", 200)
	ch3 := f.CreateChallenge(t, "getbyids_3", 300)

	ids := []uuid.UUID{ch1.ID, ch2.ID, ch3.ID}
	got, err := f.ChallengeRepo.GetByIDs(ctx, ids)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, ch1.ID, got[ch1.ID].ID)
	assert.Equal(t, 100, got[ch1.ID].Points)
	assert.Equal(t, ch2.ID, got[ch2.ID].ID)
	assert.Equal(t, 200, got[ch2.ID].Points)
	assert.Equal(t, ch3.ID, got[ch3.ID].ID)
	assert.Equal(t, 300, got[ch3.ID].Points)
}

func TestChallengeRepo_GetByIDs_EmptyIDs(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	got, err := f.ChallengeRepo.GetByIDs(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = f.ChallengeRepo.GetByIDs(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestChallengeRepo_GetByIDs_PartialMatch(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "partial_1", 50)
	nonExistentID := uuid.New()

	ids := []uuid.UUID{ch1.ID, nonExistentID}
	got, err := f.ChallengeRepo.GetByIDs(ctx, ids)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, ch1.ID, got[ch1.ID].ID)
	assert.Equal(t, 50, got[ch1.ID].Points)
}

func TestChallengeRepo_MetadataRoundTrip(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "metadata_1", 100)
	ch2 := f.CreateChallenge(t, "metadata_2", 200)

	ch1.Attribution = "Author"
	ch1.NextChallengeID = &ch2.ID
	require.NoError(t, f.ChallengeRepo.Update(ctx, ch1))

	got, err := f.ChallengeRepo.GetByID(ctx, ch1.ID)
	require.NoError(t, err)
	assert.Equal(t, "Author", got.Attribution)
	require.NotNil(t, got.NextChallengeID)
	assert.Equal(t, ch2.ID, *got.NextChallengeID)

	got.NextChallengeID = nil
	require.NoError(t, f.ChallengeRepo.Update(ctx, got))

	got, err = f.ChallengeRepo.GetByID(ctx, ch1.ID)
	require.NoError(t, err)
	assert.Nil(t, got.NextChallengeID)
}
