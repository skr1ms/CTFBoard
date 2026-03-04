package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHintRepo_GetByID_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "hint_get", 100)
	hint := &entity.Hint{ChallengeID: challenge.ID, Content: "Hint", Cost: 10, OrderIndex: 0}
	err := f.HintRepo.Create(ctx, hint)
	require.NoError(t, err)

	got, err := f.HintRepo.GetByID(ctx, hint.ID)
	require.NoError(t, err)
	assert.Equal(t, hint.ID, got.ID)
	assert.Equal(t, "Hint", got.Content)
}

func TestHintRepo_GetByID_Error(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.HintRepo.GetByID(ctx, uuid.Nil)
	assert.Error(t, err)
}

func TestHintRepo_Create_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	challenge := f.CreateChallenge(t, "hint_create_err", 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	hint := &entity.Hint{ChallengeID: challenge.ID, Content: "x", Cost: 0, OrderIndex: 0}
	err := f.HintRepo.Create(ctx, hint)
	assert.Error(t, err)
}

func TestHintRepo_Update_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	hint := f.CreateHint(t, f.CreateChallenge(t, "hint_upd_err", 100).ID, 0, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	hint.Content = "updated"
	err := f.HintRepo.Update(ctx, hint)
	assert.Error(t, err)
}

func TestHintRepo_Delete_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	hint := f.CreateHint(t, f.CreateChallenge(t, "hint_del_err", 100).ID, 0, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.HintRepo.Delete(ctx, hint.ID)
	assert.Error(t, err)
}

func TestHintRepo_CRUD_Success(t *testing.T) {
	t.Parallel()
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

func TestHintUnlockRepo_Flow(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "u1")
	challenge := f.CreateChallenge(t, "C1", 100)
	hint := f.CreateHint(t, challenge.ID, 10, 1)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.HintRepo.CreateUnlock(txCtx, team.ID, hint.ID)
	})
	require.NoError(t, err)

	unlock, err := f.HintRepo.GetByTeamAndHint(ctx, team.ID, hint.ID)
	require.NoError(t, err)
	assert.Equal(t, team.ID, unlock.TeamID)
	assert.Equal(t, hint.ID, unlock.HintID)

	IDs, err := f.HintRepo.GetUnlockedHintIDs(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Contains(t, IDs, hint.ID)
}

func TestAwardRepo_CreateTx_And_Total_InHintTest(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "u2")

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		award := &entity.Award{TeamID: team.ID, Value: -50, Description: "Hint penalty"}
		return f.AwardRepo.Create(txCtx, award)
	})
	require.NoError(t, err)
	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, -50, total)

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		award := &entity.Award{TeamID: team.ID, Value: 100, Description: "Bonus"}
		return f.AwardRepo.Create(txCtx, award)
	})
	require.NoError(t, err)

	total, err = f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, total)
}

func TestHintUnlockRepo_Rollback_UnlockNotPersistedOnError(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "unlock_rb")
	challenge := f.CreateChallenge(t, "C_unlock_rb", 100)
	hint := f.CreateHint(t, challenge.ID, 20, 1)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		if innerErr := f.HintRepo.CreateUnlock(txCtx, team.ID, hint.ID); innerErr != nil {
			return innerErr
		}
		return errors.New("forced rollback")
	})
	require.Error(t, err)

	ids, err := f.HintRepo.GetUnlockedHintIDs(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.NotContains(t, ids, hint.ID, "hint unlock should be rolled back")
}

func TestHintUnlockRepo_Rollback_AwardOrphan(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "award_rb")
	challenge := f.CreateChallenge(t, "C_award_rb", 100)
	hint := f.CreateHint(t, challenge.ID, 30, 1)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		award := &entity.Award{TeamID: team.ID, Value: -30, Description: "hint cost"}
		if innerErr := f.AwardRepo.Create(txCtx, award); innerErr != nil {
			return innerErr
		}
		if innerErr := f.HintRepo.CreateUnlock(txCtx, team.ID, hint.ID); innerErr != nil {
			return innerErr
		}
		return errors.New("forced rollback after both writes")
	})
	require.Error(t, err)

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, total, "hint penalty award should be rolled back")

	ids, err := f.HintRepo.GetUnlockedHintIDs(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.NotContains(t, ids, hint.ID, "hint unlock should be rolled back")
}

func TestHintUnlockAndAwardTx_Commit(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "award_commit")
	challenge := f.CreateChallenge(t, "C_award_commit", 100)
	hint := f.CreateHint(t, challenge.ID, 25, 1)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		award := &entity.Award{TeamID: team.ID, Value: -25, Description: "hint cost"}
		if innerErr := f.AwardRepo.Create(txCtx, award); innerErr != nil {
			return innerErr
		}
		return f.HintRepo.CreateUnlock(txCtx, team.ID, hint.ID)
	})
	require.NoError(t, err)

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, -25, total)

	ids, err := f.HintRepo.GetUnlockedHintIDs(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Contains(t, ids, hint.ID)
}

func TestScoreboardWithAwards(t *testing.T) {
	t.Parallel()
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

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		award := &entity.Award{TeamID: team.ID, Value: -20, Description: "Penalty"}
		return f.AwardRepo.Create(txCtx, award)
	})
	require.NoError(t, err)

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
