package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create Tests

func TestUserRepo_Create(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := &entity.User{
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateUser(t, "duplicate")

	user2 := &entity.User{
		Username:     "user_duplicate",
		Email:        "test2@example.com",
		PasswordHash: "hash456",
	}

	err := f.UserRepo.Create(ctx, user2)
	assert.Error(t, err)
}

func TestUserRepo_Create_DuplicateEmail(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateUser(t, "user1")

	user2 := &entity.User{
		Username:     "user2",
		Email:        "user_user1@x.com",
		PasswordHash: "hash456",
	}

	err := f.UserRepo.Create(ctx, user2)
	assert.Error(t, err)
}

// GetByID Tests

func TestUserRepo_GetByID(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	nonExistentID := uuid.New()
	_, err := f.UserRepo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

// GetByEmail Tests

func TestUserRepo_GetByEmail(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.UserRepo.GetByEmail(ctx, "nonexistent@example.com")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

// GetByUsername Tests

func TestUserRepo_GetByUsername(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.UserRepo.GetByUsername(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

// GetAll Tests

func TestUserRepo_GetAll_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1 := f.CreateUser(t, "get_all_1")
	u2 := f.CreateUser(t, "get_all_2")

	users, err := f.UserRepo.GetAll(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 2)

	ids := make(map[uuid.UUID]bool)
	for _, u := range users {
		ids[u.ID] = true
	}
	assert.True(t, ids[u1.ID])
	assert.True(t, ids[u2.ID])
}

func TestUserRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	users, err := f.UserRepo.GetAll(ctx)
	assert.Error(t, err)
	assert.Nil(t, users)
}

// GetByTeamID Tests

func TestUserRepo_GetByTeamID(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "empty_team")

	members, err := f.UserRepo.GetByTeamID(ctx, team.ID)
	require.NoError(t, err)
	assert.Len(t, members, 0)
}

// UpdateTeamID Tests
func TestUserRepo_UpdateTeamID(t *testing.T) {
	t.Helper()
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
	t.Helper()
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

// Role Persistence Tests

func TestUserRepo_Role_Persistence(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	repo := persistent.NewUserRepo(testPool.Pool)
	ctx := context.Background()

	user := &entity.User{
		Username:     "roleuser",
		Email:        "roleuser@example.com",
		PasswordHash: "hash123",
		Role:         entity.RoleAdmin,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, entity.RoleAdmin, gotUser.Role)
}
