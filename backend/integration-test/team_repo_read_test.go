package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

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
