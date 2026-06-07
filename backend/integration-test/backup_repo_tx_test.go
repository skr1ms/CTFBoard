package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestBackupRepo_TM_Run_FullImport_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "full_import")
	challenge := f.CreateChallenge(t, "FullImport", 200)
	f.AddUserToTeam(t, user.ID, team.ID)

	data := f.NewMinimalBackupData(t)
	comp, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)

	data.Competition = comp
	data.Challenges = []domain.ChallengeExport{
		{Challenge: *challenge, Hints: []domain.Hint{}},
	}
	data.Teams = []domain.TeamExport{
		{Team: *team, InviteToken: team.InviteToken, InviteTokenExpiresAt: team.InviteTokenExpiresAt, MemberIDs: []uuid.UUID{user.ID}},
	}
	data.Users = []domain.UserExport{
		{ID: user.ID, Username: user.Username, Email: user.Email, Role: string(user.Role), TeamID: &team.ID},
	}
	opts := domain.ImportOptions{ConflictMode: domain.ConflictModeOverwrite}

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		err := f.BackupRepo.ImportCompetition(txCtx, data.Competition)
		if err != nil {
			return err
		}

		err = f.BackupRepo.ImportChallenges(txCtx, data)
		if err != nil {
			return err
		}

		err = f.BackupRepo.ImportTeams(txCtx, data, opts)
		if err != nil {
			return err
		}

		err = f.BackupRepo.ImportUsers(txCtx, data, opts)
		if err != nil {
			return err
		}

		err = f.BackupRepo.ImportAwards(txCtx, data)
		if err != nil {
			return err
		}

		err = f.BackupRepo.ImportSolves(txCtx, data)
		if err != nil {
			return err
		}

		return f.BackupRepo.ImportFileMetadata(txCtx, data)
	})

	require.NoError(t, err)

	gotComp, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, data.Competition.Name, gotComp.Name)
}
