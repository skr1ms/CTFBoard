package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create Tests

func TestChallengeRepo_Create(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := &entity.Challenge{
		Title:        "Test Challenge",
		Description:  "Test Description",
		Category:     "Web",
		Points:       100,
		FlagHash:     "hash123",
		IsHidden:     false,
		InitialValue: 100,
		MinValue:     50,
		Decay:        10,
	}

	err := f.ChallengeRepo.Create(ctx, challenge)
	require.NoError(t, err)
	assert.NotEmpty(t, challenge.Id)
}

// GetByID Tests

func TestChallengeRepo_GetByID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "get_by_id", 200, 100, 20)

	gotChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, challenge.Id, gotChallenge.Id)
	assert.Equal(t, challenge.Title, gotChallenge.Title)
	assert.Equal(t, challenge.Points, gotChallenge.Points)
	assert.Equal(t, challenge.FlagHash, gotChallenge.FlagHash)
	assert.Equal(t, challenge.InitialValue, gotChallenge.InitialValue)
	assert.Equal(t, challenge.MinValue, gotChallenge.MinValue)
	assert.Equal(t, challenge.Decay, gotChallenge.Decay)
}

func TestChallengeRepo_GetByID_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	nonExistentID := uuid.New()
	_, err := f.ChallengeRepo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
}

// GetAll Tests

func TestChallengeRepo_GetAll_NoTeam(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateChallenge(t, "public_1", 100)
	f.CreateChallenge(t, "public_2", 200)

	hiddenChallenge := &entity.Challenge{
		Title:       "Hidden Challenge",
		Description: "Description",
		Category:    "Pwn",
		Points:      300,
		FlagHash:    "hash3",
		IsHidden:    true,
	}
	err := f.ChallengeRepo.Create(ctx, hiddenChallenge)
	require.NoError(t, err)

	challenges, err := f.ChallengeRepo.GetAll(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, challenges, 2)
	for _, ch := range challenges {
		assert.False(t, ch.Challenge.IsHidden)
		assert.False(t, ch.Solved)
	}
}

func TestChallengeRepo_GetAll_WithTeam(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "team_user")

	err := f.UserRepo.UpdateTeamId(ctx, user.Id, &team.Id)
	require.NoError(t, err)

	ch1 := f.CreateChallenge(t, "ch_1", 100)
	f.CreateChallenge(t, "ch_2", 200)

	f.CreateSolve(t, user.Id, team.Id, ch1.Id)

	challenges, err := f.ChallengeRepo.GetAll(ctx, &team.Id)
	require.NoError(t, err)
	assert.Len(t, challenges, 2)

	solved := false
	for _, ch := range challenges {
		if ch.Challenge.Id == ch1.Id {
			assert.True(t, ch.Solved)
			solved = true
		} else {
			assert.False(t, ch.Solved)
		}
	}
	assert.True(t, solved)
}

// Update Tests

func TestChallengeRepo_Update(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "original", 100, 50, 10)

	challenge.Title = "Updated Title"
	challenge.Description = "Updated Description"
	challenge.Category = "Crypto"
	challenge.Points = 200
	challenge.FlagHash = "updated_hash"
	challenge.IsHidden = true
	challenge.InitialValue = 200
	challenge.MinValue = 80
	challenge.Decay = 15

	err := f.ChallengeRepo.Update(ctx, challenge)
	require.NoError(t, err)

	gotChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", gotChallenge.Title)
	assert.Equal(t, "Updated Description", gotChallenge.Description)
	assert.Equal(t, "Crypto", gotChallenge.Category)
	assert.Equal(t, 200, gotChallenge.Points)
	assert.Equal(t, "updated_hash", gotChallenge.FlagHash)
	assert.True(t, gotChallenge.IsHidden)
	assert.Equal(t, 200, gotChallenge.InitialValue)
	assert.Equal(t, 80, gotChallenge.MinValue)
	assert.Equal(t, 15, gotChallenge.Decay)
}

func TestChallengeRepo_Delete(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "to_delete", 100)

	err := f.ChallengeRepo.Delete(ctx, challenge.Id)
	require.NoError(t, err)

	_, err = f.ChallengeRepo.GetByID(ctx, challenge.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
}

func TestChallengeRepo_GetByIDTx(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "tx_get", 200, 100, 20)
	_, err := f.Pool.Exec(ctx, "UPDATE challenges SET solve_count = 5 WHERE id = $1", challenge.Id)
	require.NoError(t, err)
	challenge.SolveCount = 5

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	gotChallenge, err := f.TxRepo.GetChallengeByIDTx(ctx, tx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, challenge.Id, gotChallenge.Id)
	assert.Equal(t, challenge.Title, gotChallenge.Title)
	assert.Equal(t, challenge.Points, gotChallenge.Points)
	assert.Equal(t, challenge.SolveCount, gotChallenge.SolveCount)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestChallengeRepo_GetByIDTx_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	nonExistentID := uuid.New()
	_, err = f.TxRepo.GetChallengeByIDTx(ctx, tx, nonExistentID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
}

func TestChallengeRepo_IncrementSolveCountTx(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "inc_solve", 100, 50, 10)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	_, err = f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, challenge.Id)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	tx2, _ := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	gotChallenge, err := f.TxRepo.GetChallengeByIDTx(ctx, tx2, challenge.Id)
	require.NoError(t, err)
	_ = tx2.Rollback(ctx)
	assert.Equal(t, 1, gotChallenge.SolveCount)
}

func TestChallengeRepo_UpdatePointsTx(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "update_pts", 500, 100, 10)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	newPoints := 350
	err = f.TxRepo.UpdateChallengePointsTx(ctx, tx, challenge.Id, newPoints)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	tx2, _ := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	gotChallenge, err := f.TxRepo.GetChallengeByIDTx(ctx, tx2, challenge.Id)
	require.NoError(t, err)
	_ = tx2.Rollback(ctx)
	assert.Equal(t, newPoints, gotChallenge.Points)
}

func TestChallengeRepo_AtomicDynamicScoring(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	initialValue := 500
	minValue := 100
	decay := 10

	challenge := f.CreateDynamicChallenge(t, "atomic_scoring", initialValue, minValue, decay)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	gotChallenge, err := f.TxRepo.GetChallengeByIDTx(ctx, tx, challenge.Id)
	require.NoError(t, err)

	solveCount := gotChallenge.SolveCount + 1
	newPoints := int(float64(gotChallenge.MinValue) + (float64(gotChallenge.InitialValue-gotChallenge.MinValue) / (1 + float64(solveCount-1)/float64(gotChallenge.Decay))))
	if newPoints < gotChallenge.MinValue {
		newPoints = gotChallenge.MinValue
	}

	_, err = f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, challenge.Id)
	require.NoError(t, err)

	err = f.TxRepo.UpdateChallengePointsTx(ctx, tx, challenge.Id, newPoints)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	finalChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, finalChallenge.SolveCount)
	assert.Equal(t, newPoints, finalChallenge.Points)
	// First solve check: 100 + 400/(1+0/10) = 500. Points haven't dropped yet.
	assert.Equal(t, initialValue, finalChallenge.Points)
}
