package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestTeamRepo_Create_DuplicateName(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team1 := f.CreateUserWithTeam(t, "duplicate_1")

	user2 := f.CreateUser(t, "duplicate_2")

	team2 := &domain.Team{
		Name:        team1.Name,
		InviteToken: uuid.New(),
		CaptainID:   user2.ID,
	}
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.TeamRepo.Create(ctx, team2)
	})
	assert.Error(t, err)

	gotTeam1, err := f.TeamRepo.GetByName(ctx, team1.Name)
	require.NoError(t, err)
	assert.Equal(t, team1.InviteToken, gotTeam1.InviteToken)
}

func TestTeamRepo_Create_Solo(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "solo_repo")

	team := &domain.Team{
		Name:          "SoloRepo",
		InviteToken:   uuid.New(),
		CaptainID:     user.ID,
		IsSolo:        true,
		IsAutoCreated: false,
	}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.TeamRepo.Create(txCtx, team)
	})
	require.NoError(t, err)

	gotTeam, err := f.TeamRepo.GetByID(ctx, team.ID)
	require.NoError(t, err)
	assert.True(t, gotTeam.IsSolo)
	assert.False(t, gotTeam.IsAutoCreated)
}

func TestTeamRepo_Ban_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "ban_success")

	err := f.TeamRepo.Ban(ctx, team.ID, "rule violation")
	require.NoError(t, err)

	gotTeam, err := f.TeamRepo.GetByID(ctx, team.ID)
	require.NoError(t, err)
	assert.True(t, gotTeam.IsBanned)
	assert.NotNil(t, gotTeam.BannedAt)
	require.NotNil(t, gotTeam.BannedReason)
	assert.Equal(t, "rule violation", *gotTeam.BannedReason)
}

func TestTeamRepo_Ban_Error_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TeamRepo.Ban(ctx, uuid.New(), "reason")
	assert.Error(t, err)
}

func TestTeamRepo_Unban_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "unban_success")
	err := f.TeamRepo.Ban(ctx, team.ID, "reason")
	require.NoError(t, err)

	err = f.TeamRepo.Unban(ctx, team.ID)
	require.NoError(t, err)

	gotTeam, err := f.TeamRepo.GetByID(ctx, team.ID)
	require.NoError(t, err)
	assert.False(t, gotTeam.IsBanned)
	assert.Nil(t, gotTeam.BannedAt)
	assert.Nil(t, gotTeam.BannedReason)
}

func TestTeamRepo_Unban_Error_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TeamRepo.Unban(ctx, uuid.New())
	assert.Error(t, err)
}

func TestTeamRepo_SetHidden_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "set_hidden_success")

	err := f.TeamRepo.SetHidden(ctx, team.ID, true)
	require.NoError(t, err)

	gotTeam, err := f.TeamRepo.GetByID(ctx, team.ID)
	require.NoError(t, err)
	assert.True(t, gotTeam.IsHidden)

	err = f.TeamRepo.SetHidden(ctx, team.ID, false)
	require.NoError(t, err)

	gotTeam, err = f.TeamRepo.GetByID(ctx, team.ID)
	require.NoError(t, err)
	assert.False(t, gotTeam.IsHidden)
}

func TestTeamRepo_SetHidden_Error_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TeamRepo.SetHidden(ctx, uuid.New(), true)
	assert.Error(t, err)
}

func TestTeamRepo_HardDeleteTeams_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "hard_del_success")
	err := f.TeamRepo.Delete(ctx, team.ID)
	require.NoError(t, err)

	cutoff := time.Now().Add(-1 * time.Hour)
	f.BackdateTeamDeletedAt(t, team.ID, cutoff)

	err = f.TeamRepo.HardDeleteTeams(ctx, time.Now().Add(-30*time.Minute))
	require.NoError(t, err)

	_, err = f.TeamRepo.GetByID(ctx, team.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
}

func TestTeamRepo_CreateAuditLog_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "audit_log_success")

	log := &domain.TeamAuditLog{
		TeamID: team.ID,
		UserID: &team.CaptainID,
		Action: domain.TeamActionCreated,
	}
	err := f.TeamRepo.CreateAuditLog(ctx, log)
	require.NoError(t, err)
	require.NotEmpty(t, log.ID)
	require.False(t, log.CreatedAt.IsZero())
}
