package integration_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Hint CRUD Tests

func TestHintRepo_CRUD(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "hint_crud", 100)

	hint := &entity.Hint{
		ChallengeID: challenge.ID,
		Content:     "Secret Hint",
		Cost:        50,
		OrderIndex:  1,
	}
	err := f.HintRepo.Create(ctx, hint)
	require.NoError(t, err)
	assert.NotEmpty(t, hint.ID)

	gotHint, err := f.HintRepo.GetByID(ctx, hint.ID)
	require.NoError(t, err)
	assert.Equal(t, hint.Content, gotHint.Content)
	assert.Equal(t, hint.Cost, gotHint.Cost)

	hint.Content = "Updated Hint"
	hint.Cost = 75
	err = f.HintRepo.Update(ctx, hint)
	require.NoError(t, err)

	gotHintUpdated, err := f.HintRepo.GetByID(ctx, hint.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Hint", gotHintUpdated.Content)
	assert.Equal(t, 75, gotHintUpdated.Cost)

	err = f.HintRepo.Delete(ctx, hint.ID)
	require.NoError(t, err)

	_, err = f.HintRepo.GetByID(ctx, hint.ID)
	assert.Error(t, err)
}

// HintUnlock Tests

func TestHintUnlockRepo_Flow(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "u1")
	challenge := f.CreateChallenge(t, "C1", 100)
	hint := f.CreateHint(t, challenge.ID, 10, 1)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	err = f.TxRepo.CreateHintUnlockTx(ctx, tx, team.ID, hint.ID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	unlock, err := f.HintUnlockRepo.GetByTeamAndHint(ctx, team.ID, hint.ID)
	require.NoError(t, err)
	assert.Equal(t, team.ID, unlock.TeamID)
	assert.Equal(t, hint.ID, unlock.HintID)

	IDs, err := f.HintUnlockRepo.GetUnlockedHintIDs(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Contains(t, IDs, hint.ID)
}

// Award Tests (in HintTest file)

func TestAwardRepo_CreateTx_And_Total_InHintTest(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "u2")

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	f.CreateAwardTx(t, tx, team.ID, -50, "Hint penalty")
	require.NoError(t, tx.Commit(ctx))
	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, -50, total)

	tx2, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	f.CreateAwardTx(t, tx2, team.ID, 100, "Bonus")
	require.NoError(t, tx2.Commit(ctx))

	total, err = f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, total) // -50 + 100 = 50
}

func TestScoreboardWithAwards(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "u3")

	err := f.UserRepo.UpdateTeamID(ctx, user.ID, &team.ID)
	require.NoError(t, err)

	challenge := f.CreateChallenge(t, "C3", 100)

	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	score, err := f.SolveRepo.GetTeamScore(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 100, score)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	f.CreateAwardTx(t, tx, team.ID, -20, "Penalty")
	require.NoError(t, tx.Commit(ctx))

	score, err = f.SolveRepo.GetTeamScore(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 80, score)

	scoreboard, err := f.SolveRepo.GetScoreboard(ctx)
	require.NoError(t, err)
	found := false
	for _, entry := range scoreboard {
		if entry.TeamID == team.ID {
			assert.Equal(t, 80, entry.Points)
			found = true
			break
		}
	}
	assert.True(t, found)
}
