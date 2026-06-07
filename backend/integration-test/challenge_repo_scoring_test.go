package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallengeRepo_IncrementSolveCountTx(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "inc_solve", 100, 50, 10)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		_, err := f.ChallengeRepo.IncrementSolveCount(txCtx, challenge.ID)

		return err
	})
	require.NoError(t, err)

	gotChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, gotChallenge.SolveCount)
}

func TestChallengeRepo_UpdatePointsTx(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "update_pts", 500, 100, 10)

	newPoints := 350
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.UpdatePoints(txCtx, challenge.ID, newPoints)
	})
	require.NoError(t, err)

	gotChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, newPoints, gotChallenge.Points)
}

func TestChallengeRepo_AtomicDynamicScoring(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	initialValue := 500
	minValue := 100
	decay := 10

	challenge := f.CreateDynamicChallenge(t, "atomic_scoring", initialValue, minValue, decay)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		gotChallenge, err := f.ChallengeRepo.GetByIDForUpdate(txCtx, challenge.ID)
		if err != nil {
			return err
		}

		solveCount := gotChallenge.SolveCount + 1

		newPoints := max(int(float64(gotChallenge.MinValue)+(float64(gotChallenge.InitialValue-gotChallenge.MinValue)/(1+float64(solveCount-1)/float64(gotChallenge.Decay)))), gotChallenge.MinValue)

		_, err = f.ChallengeRepo.IncrementSolveCount(txCtx, challenge.ID)
		if err != nil {
			return err
		}

		return f.ChallengeRepo.UpdatePoints(txCtx, challenge.ID, newPoints)
	})
	require.NoError(t, err)

	finalChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, finalChallenge.SolveCount)
	// For first solve the dynamic formula yields InitialValue
	assert.Equal(t, initialValue, finalChallenge.Points)
}

func TestChallengeRepo_BatchDecrementSolveCount(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "dec1", 100)
	ch2 := f.CreateChallenge(t, "dec2", 100)

	// Seed solve counts to 2 each
	_, err := f.Pool.Exec(ctx, "UPDATE challenges SET solve_count = 2 WHERE id = ANY($1)", []uuid.UUID{ch1.ID, ch2.ID})
	require.NoError(t, err)

	err = f.ChallengeRepo.BatchDecrementSolveCount(ctx, []uuid.UUID{ch1.ID, ch2.ID})
	require.NoError(t, err)

	got1, err := f.ChallengeRepo.GetByID(ctx, ch1.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got1.SolveCount)

	got2, err := f.ChallengeRepo.GetByID(ctx, ch2.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got2.SolveCount)
}

func TestChallengeRepo_BatchIncrementSolveCount(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "inc1", 100)
	ch2 := f.CreateChallenge(t, "inc2", 100)

	err := f.ChallengeRepo.BatchIncrementSolveCount(ctx, []uuid.UUID{ch1.ID, ch2.ID})
	require.NoError(t, err)

	got1, err := f.ChallengeRepo.GetByID(ctx, ch1.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got1.SolveCount)

	got2, err := f.ChallengeRepo.GetByID(ctx, ch2.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got2.SolveCount)
}

func TestChallengeRepo_BatchUpdatePoints(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "bup1", 100)
	ch2 := f.CreateChallenge(t, "bup2", 200)

	err := f.ChallengeRepo.BatchUpdatePoints(ctx, []uuid.UUID{ch1.ID, ch2.ID}, []int{150, 250})
	require.NoError(t, err)

	got1, err := f.ChallengeRepo.GetByID(ctx, ch1.ID)
	require.NoError(t, err)
	assert.Equal(t, 150, got1.Points)

	got2, err := f.ChallengeRepo.GetByID(ctx, ch2.ID)
	require.NoError(t, err)
	assert.Equal(t, 250, got2.Points)
}

func TestChallengeRepo_RecalculateSolveCounts(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "recalc")
	ch := f.CreateChallenge(t, "recalc_ch", 100)
	f.CreateSolve(t, user.ID, team.ID, ch.ID)

	// Corrupt the solve_count
	_, err := f.Pool.Exec(ctx, "UPDATE challenges SET solve_count = 99 WHERE id = $1", ch.ID)
	require.NoError(t, err)

	err = f.ChallengeRepo.RecalculateSolveCounts(ctx, []uuid.UUID{ch.ID})
	require.NoError(t, err)

	got, err := f.ChallengeRepo.GetByID(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.SolveCount, "RecalculateSolveCounts should fix corrupted solve_count")
}
