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

func TestTeamRepo_Create(t *testing.T) {
	t.Parallel()
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

func TestTeamRepo_GetByID(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	nonExistentID := uuid.New()
	_, err := f.TeamRepo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
}

func TestTeamRepo_GetByInviteToken(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.TeamRepo.GetByInviteToken(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
}

func TestTeamRepo_GetByName(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.TeamRepo.GetByName(ctx, "nonexistent_team")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
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

func TestTeamRepo_HardDeleteTeams_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.TeamRepo.HardDeleteTeams(ctx, time.Now())
	assert.Error(t, err)
}

func TestTeamRepo_GetAll_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team1 := f.CreateUserWithTeam(t, "get_all_1")
	_, team2 := f.CreateUserWithTeam(t, "get_all_2")

	teams, err := f.TeamRepo.GetAll(ctx)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, tm := range teams {
		ids[tm.ID] = true
	}

	assert.True(t, ids[team1.ID])
	assert.True(t, ids[team2.ID])
}

func TestTeamRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	teams, err := f.TeamRepo.GetAll(ctx)
	assert.Error(t, err)
	assert.Nil(t, teams)
}

func TestTeamRepo_CountActiveTeams_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team1 := f.CreateUserWithTeam(t, "count_active_1")
	_, team2 := f.CreateUserWithTeam(t, "count_active_2")

	count, err := f.TeamRepo.CountActiveTeams(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, count, 2)

	_ = team1
	_ = team2
}

func TestTeamRepo_CountActiveTeams_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.TeamRepo.CountActiveTeams(ctx)
	require.Error(t, err)
}

func TestTeamRepo_Lock_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "lock_success")

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.TeamRepo.Lock(txCtx, team.ID)
	})
	require.NoError(t, err)
}

func TestTeamRepo_Lock_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	_, team := f.CreateUserWithTeam(t, "lock_cancel")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.TeamRepo.Lock(ctx, team.ID)
	require.Error(t, err)
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

func TestTeamRepo_CreateAuditLog_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	_, team := f.CreateUserWithTeam(t, "audit_cancel")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	log := &domain.TeamAuditLog{
		TeamID: team.ID,
		UserID: &team.CaptainID,
		Action: domain.TeamActionCreated,
	}
	err := f.TeamRepo.CreateAuditLog(ctx, log)
	require.Error(t, err)
}

func TestTeamRepo_GetSoloTeamByUserID_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "solo_lookup")
	team := &domain.Team{
		Name:          "SoloLookup",
		InviteToken:   uuid.New(),
		CaptainID:     user.ID,
		IsSolo:        true,
		IsAutoCreated: false,
	}
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.TeamRepo.Create(txCtx, team)
	})
	require.NoError(t, err)
	err = f.UserRepo.UpdateTeamID(ctx, user.ID, &team.ID)
	require.NoError(t, err)

	got, err := f.TeamRepo.GetSoloTeamByUserID(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, team.ID, got.ID)
	assert.True(t, got.IsSolo)
}

func TestTeamRepo_GetSoloTeamByUserID_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "solo_notfound")
	_, err := f.TeamRepo.GetSoloTeamByUserID(ctx, user.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
}
