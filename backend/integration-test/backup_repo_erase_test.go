package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestBackupRepo_EraseAllTablesTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	t.Cleanup(func() { f.ResetCompetition(t) })

	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "erase_user")
	challenge := f.CreateChallenge(t, "erase_chall", 100)
	page := f.CreatePage(t, "erase_page", false)
	notification := f.CreateNotification(t, "erase_notif")

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseAllTables(txCtx)
	})
	require.NoError(t, err)

	_, err = f.UserRepo.GetByID(ctx, user.ID)
	assert.Error(t, err)
	_, err = f.TeamRepo.GetByID(ctx, team.ID)
	assert.Error(t, err)
	_, err = f.ChallengeRepo.GetByID(ctx, challenge.ID)
	assert.Error(t, err)

	gotPage, err := f.PageRepo.GetByID(ctx, page.ID)
	require.NoError(t, err)
	assert.Equal(t, page.ID, gotPage.ID)

	gotNotification, err := f.NotificationRepo.GetByID(ctx, notification.ID)
	require.NoError(t, err)
	assert.Equal(t, notification.ID, gotNotification.ID)
}

func TestBackupRepo_EraseAllTablesTx_Error_ClosedTx(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseAllTables(txCtx)
	})
	assert.Error(t, err)
}

func TestBackupRepo_TM_Run_EraseAndImportChallenges_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	t.Cleanup(func() { f.ResetCompetition(t) })

	ctx := context.Background()

	challenge := f.CreateChallenge(t, "EraseImport", 300)
	challengeID := challenge.ID

	data := f.NewMinimalBackupData(t)
	data.Competition = f.ActiveCompetition(t, "Restored")
	data.Challenges = []domain.ChallengeExport{
		{Challenge: domain.Challenge{ID: challengeID, Title: "Restored Chall", Description: challenge.Description, Category: challenge.Category, Points: 400, FlagHash: challenge.FlagHash, InitialValue: 400, MinValue: 400, Decay: 0, SolveCount: 0}, Hints: []domain.Hint{}},
	}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		err := f.BackupRepo.EraseAllTables(txCtx)
		if err != nil {
			return err
		}

		err = f.BackupRepo.ImportCompetition(txCtx, data.Competition)
		if err != nil {
			return err
		}

		return f.BackupRepo.ImportChallenges(txCtx, data)
	})

	require.NoError(t, err)

	got, err := f.ChallengeRepo.GetByID(ctx, challengeID)
	require.NoError(t, err)
	assert.Equal(t, "Restored Chall", got.Title)
	assert.Equal(t, 400, got.Points)
}

func TestBackupRepo_EraseTables_DisallowedTable_ReturnsError(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseTables(txCtx, []string{"solves", "evil_table"})
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not allowed")
	assert.Contains(t, err.Error(), "evil_table")
}

func TestBackupRepo_EraseTables_AllowedSubset_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "erase_subset")
	challenge := f.CreateChallenge(t, "erase_subset_ch", 100)
	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseTables(txCtx, []string{"solves"})
	})
	require.NoError(t, err)

	_, err = f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	assert.Error(t, err)
	_, err = f.UserRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	_, err = f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
}

func TestBackupRepo_EraseTables_EmptyTables_NoOp(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseTables(txCtx, nil)
	})
	require.NoError(t, err)

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseTables(txCtx, []string{})
	})
	require.NoError(t, err)
}
