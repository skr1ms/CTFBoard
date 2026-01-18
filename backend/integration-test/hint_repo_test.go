package integration_test

import (
	"context"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Hint CRUD Tests

func TestHintRepo_CRUD(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "hint_crud", 100)

	hint := &entity.Hint{
		ChallengeId: challenge.Id,
		Content:     "Secret Hint",
		Cost:        50,
		OrderIndex:  1,
	}
	err := f.HintRepo.Create(ctx, hint)
	require.NoError(t, err)
	assert.NotEmpty(t, hint.Id)

	gotHint, err := f.HintRepo.GetByID(ctx, hint.Id)
	require.NoError(t, err)
	assert.Equal(t, hint.Content, gotHint.Content)
	assert.Equal(t, hint.Cost, gotHint.Cost)

	hint.Content = "Updated Hint"
	hint.Cost = 75
	err = f.HintRepo.Update(ctx, hint)
	require.NoError(t, err)

	gotHintUpdated, err := f.HintRepo.GetByID(ctx, hint.Id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Hint", gotHintUpdated.Content)
	assert.Equal(t, 75, gotHintUpdated.Cost)

	err = f.HintRepo.Delete(ctx, hint.Id)
	require.NoError(t, err)

	_, err = f.HintRepo.GetByID(ctx, hint.Id)
	assert.Error(t, err)
}

// HintUnlock Tests

func TestHintUnlockRepo_Flow(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "u1")
	challenge := f.CreateChallenge(t, "C1", 100)
	hint := f.CreateHint(t, challenge.Id, 10, 1)

	tx, err := f.DB.BeginTx(ctx, nil)
	require.NoError(t, err)

	err = f.TxRepo.CreateHintUnlockTx(ctx, tx, team.Id, hint.Id)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	unlock, err := f.HintUnlockRepo.GetByTeamAndHint(ctx, team.Id, hint.Id)
	require.NoError(t, err)
	assert.Equal(t, team.Id, unlock.TeamId)
	assert.Equal(t, hint.Id, unlock.HintId)

	ids, err := f.HintUnlockRepo.GetUnlockedHintIDs(ctx, team.Id, challenge.Id)
	require.NoError(t, err)
	assert.Contains(t, ids, hint.Id)
}

// Award Tests (in HintTest file)

func TestAwardRepo_CreateTx_And_Total_InHintTest(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "u2")

	tx, err := f.DB.BeginTx(ctx, nil)
	require.NoError(t, err)

	f.CreateAwardTx(t, tx, team.Id, -50, "Hint penalty")
	require.NoError(t, tx.Commit())

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, -50, total)

	tx2, _ := f.DB.BeginTx(ctx, nil)
	f.CreateAwardTx(t, tx2, team.Id, 100, "Bonus")
	require.NoError(t, tx2.Commit())

	total, err = f.AwardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 50, total) // -50 + 100 = 50
}

func TestScoreboardWithAwards(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "u3")

	err := f.UserRepo.UpdateTeamId(ctx, user.Id, &team.Id)
	require.NoError(t, err)

	challenge := f.CreateChallenge(t, "C3", 100)

	f.CreateSolve(t, user.Id, team.Id, challenge.Id)

	score, err := f.SolveRepo.GetTeamScore(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 100, score)

	tx, _ := f.DB.BeginTx(ctx, nil)
	f.CreateAwardTx(t, tx, team.Id, -20, "Penalty")
	require.NoError(t, tx.Commit())

	score, err = f.SolveRepo.GetTeamScore(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 80, score)

	scoreboard, err := f.SolveRepo.GetScoreboard(ctx)
	require.NoError(t, err)
	found := false
	for _, entry := range scoreboard {
		if entry.TeamId == team.Id {
			assert.Equal(t, 80, entry.Points)
			found = true
			break
		}
	}
	assert.True(t, found)
}
