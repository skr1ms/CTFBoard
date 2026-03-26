package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func TestUserRepo_Create(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := &domain.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
	}

	err := f.UserRepo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.NotEmpty(t, gotUser.ID)
	user.ID = gotUser.ID
}

func TestUserRepo_Create_DuplicateUsername(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1 := f.CreateUser(t, "duplicate")

	user2 := &domain.User{
		Username:     u1.Username,
		Email:        "other@example.com",
		PasswordHash: "hash456",
	}

	err := f.UserRepo.Create(ctx, user2)
	assert.Error(t, err)
}

func TestUserRepo_Create_DuplicateEmail(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1 := f.CreateUser(t, "user1")

	user2 := &domain.User{
		Username:     "otheruser",
		Email:        u1.Email,
		PasswordHash: "hash456",
	}

	err := f.UserRepo.Create(ctx, user2)
	assert.Error(t, err)
}

func TestUserRepo_GetByID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "get_by_ID")

	gotUser, err := f.UserRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, gotUser.ID)
	assert.Equal(t, user.Username, gotUser.Username)
	assert.Equal(t, user.Email, gotUser.Email)
	assert.Nil(t, gotUser.TeamID)
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	nonExistentID := uuid.New()
	_, err := f.UserRepo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrUserNotFound)
}

func TestUserRepo_GetByEmail(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "get_by_email")

	gotUser, err := f.UserRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.ID, gotUser.ID)
	assert.Equal(t, user.Username, gotUser.Username)
	assert.Equal(t, user.Email, gotUser.Email)
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.UserRepo.GetByEmail(ctx, "nonexistent@example.com")
	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrUserNotFound)
}

func TestUserRepo_GetByUsername(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "get_by_username")

	gotUser, err := f.UserRepo.GetByUsername(ctx, user.Username)
	require.NoError(t, err)
	assert.Equal(t, user.ID, gotUser.ID)
	assert.Equal(t, user.Username, gotUser.Username)
}

func TestUserRepo_GetByUsername_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.UserRepo.GetByUsername(ctx, "nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrUserNotFound)
}

func TestUserRepo_GetAll_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1 := f.CreateUser(t, "get_all_1")
	u2 := f.CreateUser(t, "get_all_2")

	users, err := f.UserRepo.GetAll(ctx)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, u := range users {
		ids[u.ID] = true
	}

	assert.True(t, ids[u1.ID])
	assert.True(t, ids[u2.ID])
}

func TestUserRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	users, err := f.UserRepo.GetAll(ctx)
	assert.Error(t, err)
	assert.Nil(t, users)
}

func TestUserRepo_GetByTeamID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	captain, team := f.CreateUserWithTeam(t, "team_u1")

	user2 := f.CreateUser(t, "team_u2")

	err := f.UserRepo.UpdateTeamID(ctx, captain.ID, &team.ID)
	require.NoError(t, err)

	err = f.UserRepo.UpdateTeamID(ctx, user2.ID, &team.ID)
	require.NoError(t, err)

	members, err := f.UserRepo.GetByTeamID(ctx, team.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestUserRepo_GetByTeamID_Empty(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "empty")
	team := f.CreateTeam(t, "empty", user.ID)

	members, err := f.UserRepo.GetByTeamID(ctx, team.ID)
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestUserRepo_UpdateTeamID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "update_team")

	err := f.UserRepo.UpdateTeamID(ctx, user.ID, &team.ID)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.NotNil(t, gotUser.TeamID)
	assert.Equal(t, team.ID, *gotUser.TeamID)
}

func TestUserRepo_UpdateTeamID_Remove(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "remove_team")

	err := f.UserRepo.UpdateTeamID(ctx, user.ID, &team.ID)
	require.NoError(t, err)

	err = f.UserRepo.UpdateTeamID(ctx, user.ID, nil)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Nil(t, gotUser.TeamID)
}

func TestUserRepo_Role_Persistence(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	repo := persistent.NewUserRepo(testPool.Pool)
	ctx := context.Background()

	user := &domain.User{
		Username:     "roleuser",
		Email:        "roleuser@example.com",
		PasswordHash: "hash123",
		Role:         domain.RoleAdmin,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, domain.RoleAdmin, gotUser.Role)
}

func TestUserRepo_Lock_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "lock_success")

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.UserRepo.Lock(txCtx, user.ID)
	})
	require.NoError(t, err)
}

func TestUserRepo_Lock_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "lock_cancel")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.UserRepo.Lock(ctx, user.ID)
	require.Error(t, err)
}
