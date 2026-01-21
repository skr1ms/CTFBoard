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
	assert.NotEmpty(t, gotUser.Id)
	user.Id = gotUser.Id
}

func TestUserRepo_Create_DuplicateUsername(t *testing.T) {
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
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateUser(t, "user1")

	user2 := &entity.User{
		Username:     "user2",
		Email:        "user_user1@example.com",
		PasswordHash: "hash456",
	}

	err := f.UserRepo.Create(ctx, user2)
	assert.Error(t, err)
}

// GetByID Tests

func TestUserRepo_GetByID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "get_by_id")

	gotUser, err := f.UserRepo.GetByID(ctx, user.Id)
	require.NoError(t, err)
	assert.Equal(t, user.Id, gotUser.Id)
	assert.Equal(t, user.Username, gotUser.Username)
	assert.Equal(t, user.Email, gotUser.Email)
	assert.Nil(t, gotUser.TeamId)
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
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
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "get_by_email")

	gotUser, err := f.UserRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.Id, gotUser.Id)
	assert.Equal(t, user.Username, gotUser.Username)
	assert.Equal(t, user.Email, gotUser.Email)
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.UserRepo.GetByEmail(ctx, "nonexistent@example.com")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

// GetByUsername Tests

func TestUserRepo_GetByUsername(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "get_by_username")

	gotUser, err := f.UserRepo.GetByUsername(ctx, user.Username)
	require.NoError(t, err)
	assert.Equal(t, user.Id, gotUser.Id)
	assert.Equal(t, user.Username, gotUser.Username)
}

func TestUserRepo_GetByUsername_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.UserRepo.GetByUsername(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

// GetByTeamId Tests

func TestUserRepo_GetByTeamId(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	captain, team := f.CreateUserWithTeam(t, "team_u1")

	user2 := f.CreateUser(t, "team_u2")

	err := f.UserRepo.UpdateTeamId(ctx, captain.Id, &team.Id)
	require.NoError(t, err)

	err = f.UserRepo.UpdateTeamId(ctx, user2.Id, &team.Id)
	require.NoError(t, err)

	members, err := f.UserRepo.GetByTeamId(ctx, team.Id)
	require.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestUserRepo_GetByTeamId_Empty(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "empty_team")

	members, err := f.UserRepo.GetByTeamId(ctx, team.Id)
	require.NoError(t, err)
	assert.Len(t, members, 0)
}

// UpdateTeamId Tests
func TestUserRepo_UpdateTeamId(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "update_team")

	err := f.UserRepo.UpdateTeamId(ctx, user.Id, &team.Id)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByID(ctx, user.Id)
	require.NoError(t, err)
	assert.NotNil(t, gotUser.TeamId)
	assert.Equal(t, team.Id, *gotUser.TeamId)
}

func TestUserRepo_UpdateTeamId_Remove(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "remove_team")

	err := f.UserRepo.UpdateTeamId(ctx, user.Id, &team.Id)
	require.NoError(t, err)

	err = f.UserRepo.UpdateTeamId(ctx, user.Id, nil)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByID(ctx, user.Id)
	require.NoError(t, err)
	assert.Nil(t, gotUser.TeamId)
}

// Role Persistence Tests

func TestUserRepo_Role_Persistence(t *testing.T) {
	testPool := SetupTestPool(t)
	repo := persistent.NewUserRepo(testPool.Pool)
	ctx := context.Background()

	user := &entity.User{
		Username:     "roleuser",
		Email:        "roleuser@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, "admin", gotUser.Role)
}
