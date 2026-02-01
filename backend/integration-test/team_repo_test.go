package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create Tests

func TestTeamRepo_Create(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "captain")

	assert.NotEmpty(t, team.ID)
	gotTeam, err := f.TeamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	assert.NotZero(t, gotTeam.CreatedAt)
}

func TestTeamRepo_Create_DuplicateName(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team1 := f.CreateUserWithTeam(t, "duplicate_1")

	user2 := f.CreateUser(t, "duplicate_2")

	team2 := &entity.Team{
		Name:        team1.Name,
		InviteToken: uuid.New(),
		CaptainID:   user2.ID,
	}
	err := f.TeamRepo.Create(ctx, team2)
	assert.Error(t, err)

	gotTeam1, err := f.TeamRepo.GetByName(ctx, team1.Name)
	require.NoError(t, err)
	assert.Equal(t, team1.InviteToken, gotTeam1.InviteToken)
}

// GetByID Tests

func TestTeamRepo_GetByID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_by_ID")

	gotTeam, err := f.TeamRepo.GetByID(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, team.ID, gotTeam.ID)
	assert.Equal(t, team.Name, gotTeam.Name)
	assert.Equal(t, team.InviteToken, gotTeam.InviteToken)
	assert.Equal(t, team.CaptainID, gotTeam.CaptainID)
}

func TestTeamRepo_GetByID_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	nonExistentID := uuid.New()
	_, err := f.TeamRepo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// GetByInviteToken Tests

func TestTeamRepo_GetByInviteToken(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "invite_token")

	gotTeam, err := f.TeamRepo.GetByInviteToken(ctx, team.InviteToken)
	require.NoError(t, err)
	assert.Equal(t, team.ID, gotTeam.ID)
	assert.Equal(t, team.InviteToken, gotTeam.InviteToken)
}

func TestTeamRepo_GetByInviteToken_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.TeamRepo.GetByInviteToken(ctx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// GetByName Tests

func TestTeamRepo_GetByName(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_by_name")

	gotTeam, err := f.TeamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	assert.Equal(t, team.ID, gotTeam.ID)
	assert.Equal(t, team.Name, gotTeam.Name)
}

func TestTeamRepo_GetByName_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.TeamRepo.GetByName(ctx, "nonexistent_team")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

func TestTeamRepo_Create_Solo(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "solo_repo")

	team := &entity.Team{
		Name:          "SoloRepo",
		InviteToken:   uuid.New(),
		CaptainID:     user.ID,
		IsSolo:        true,
		IsAutoCreated: false,
	}

	err := f.TeamRepo.Create(ctx, team)
	require.NoError(t, err)

	gotTeam, err := f.TeamRepo.GetByID(ctx, team.ID)
	require.NoError(t, err)
	assert.True(t, gotTeam.IsSolo)
	assert.False(t, gotTeam.IsAutoCreated)
}

func TestTeamRepo_Ban_Success(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TeamRepo.Ban(ctx, uuid.New(), "reason")
	assert.Error(t, err)
}

func TestTeamRepo_Unban_Success(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TeamRepo.Unban(ctx, uuid.New())
	assert.Error(t, err)
}

func TestTeamRepo_SetHidden_Success(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TeamRepo.SetHidden(ctx, uuid.New(), true)
	assert.Error(t, err)
}

func TestTeamRepo_HardDeleteTeams_Success(t *testing.T) {
	t.Helper()
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
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

func TestTeamRepo_HardDeleteTeams_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.TeamRepo.HardDeleteTeams(ctx, time.Now())
	assert.Error(t, err)
}

func TestTeamRepo_GetAll_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team1 := f.CreateUserWithTeam(t, "get_all_1")
	_, team2 := f.CreateUserWithTeam(t, "get_all_2")

	teams, err := f.TeamRepo.GetAll(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(teams), 2)

	ids := make(map[uuid.UUID]bool)
	for _, tm := range teams {
		ids[tm.ID] = true
	}
	assert.True(t, ids[team1.ID])
	assert.True(t, ids[team2.ID])
}

func TestTeamRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	teams, err := f.TeamRepo.GetAll(ctx)
	assert.Error(t, err)
	assert.Nil(t, teams)
}
