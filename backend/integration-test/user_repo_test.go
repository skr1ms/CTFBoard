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

func TestUserRepo_Create(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.NotEmpty(t, gotUser.Id)
	user.Id = gotUser.Id
}

func TestUserRepo_Create_DuplicateUsername(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	user1 := &entity.User{
		Username:     "duplicate",
		Email:        "test1@example.com",
		PasswordHash: "hash123",
	}

	err := repo.Create(ctx, user1)
	require.NoError(t, err)

	user2 := &entity.User{
		Username:     "duplicate",
		Email:        "test2@example.com",
		PasswordHash: "hash456",
	}

	err = repo.Create(ctx, user2)
	assert.Error(t, err)
}

func TestUserRepo_Create_DuplicateEmail(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	user1 := &entity.User{
		Username:     "user1",
		Email:        "duplicate@example.com",
		PasswordHash: "hash123",
	}

	err := repo.Create(ctx, user1)
	require.NoError(t, err)

	user2 := &entity.User{
		Username:     "user2",
		Email:        "duplicate@example.com",
		PasswordHash: "hash456",
	}

	err = repo.Create(ctx, user2)
	assert.Error(t, err)
}

func TestUserRepo_GetByID(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "getbyid",
		Email:        "getbyid@example.com",
		PasswordHash: "hash123",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id
	assert.Equal(t, user.Username, gotUser.Username)
	assert.Equal(t, user.Email, gotUser.Email)
	assert.Nil(t, gotUser.TeamId)

	gotUser2, err := repo.GetByID(ctx, user.Id)
	require.NoError(t, err)
	assert.Equal(t, user.Id, gotUser2.Id)
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	nonExistentID := uuid.New().String()
	_, err := repo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

func TestUserRepo_GetByEmail(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "getbyemail",
		Email:        "getbyemail@example.com",
		PasswordHash: "hash123",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id
	assert.Equal(t, user.Username, gotUser.Username)
	assert.Equal(t, user.Email, gotUser.Email)
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nonexistent@example.com")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

func TestUserRepo_GetByUsername(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "getbyusername",
		Email:        "getbyusername@example.com",
		PasswordHash: "hash123",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := repo.GetByUsername(ctx, user.Username)
	require.NoError(t, err)
	user.Id = gotUser.Id
	assert.Equal(t, user.Username, gotUser.Username)
	assert.Equal(t, user.Email, gotUser.Email)
}

func TestUserRepo_GetByUsername_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	_, err := repo.GetByUsername(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

func TestUserRepo_GetByTeamId(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	user1 := &entity.User{
		Username:     "teamuser1",
		Email:        "teamuser1@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user1)
	require.NoError(t, err)
	gotUser1, err := userRepo.GetByEmail(ctx, user1.Email)
	require.NoError(t, err)
	user1.Id = gotUser1.Id

	user2 := &entity.User{
		Username:     "teamuser2",
		Email:        "teamuser2@example.com",
		PasswordHash: "hash456",
	}
	err = userRepo.Create(ctx, user2)
	require.NoError(t, err)
	gotUser2, err := userRepo.GetByEmail(ctx, user2.Email)
	require.NoError(t, err)
	user2.Id = gotUser2.Id

	team := &entity.Team{
		Name:        "testteam",
		InviteToken: "token123",
		CaptainId:   user1.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	err = userRepo.UpdateTeamId(ctx, user1.Id, &team.Id)
	require.NoError(t, err)

	err = userRepo.UpdateTeamId(ctx, user2.Id, &team.Id)
	require.NoError(t, err)

	members, err := userRepo.GetByTeamId(ctx, team.Id)
	require.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestUserRepo_GetByTeamId_Empty(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "captain",
		Email:        "captain@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "emptyteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	members, err := userRepo.GetByTeamId(ctx, team.Id)
	require.NoError(t, err)
	assert.Len(t, members, 0)
}

func TestUserRepo_UpdateTeamId(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "updateteam",
		Email:        "updateteam@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "updateteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	err = userRepo.UpdateTeamId(ctx, user.Id, &team.Id)
	require.NoError(t, err)

	gotUser, err = userRepo.GetByID(ctx, user.Id)
	require.NoError(t, err)
	assert.NotNil(t, gotUser.TeamId)
	assert.Equal(t, team.Id, *gotUser.TeamId)
}

func TestUserRepo_UpdateTeamId_Remove(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "removeteam",
		Email:        "removeteam@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "removeteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	err = userRepo.UpdateTeamId(ctx, user.Id, &team.Id)
	require.NoError(t, err)

	err = userRepo.UpdateTeamId(ctx, user.Id, nil)
	require.NoError(t, err)

	gotUser, err = userRepo.GetByID(ctx, user.Id)
	require.NoError(t, err)
	assert.Nil(t, gotUser.TeamId)
}
