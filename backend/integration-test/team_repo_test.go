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

func TestTeamRepo_Create(t *testing.T) {
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
		Name:        "TestTeam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}

	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)

	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	assert.NotEmpty(t, gotTeam.Id)
	assert.NotZero(t, gotTeam.CreatedAt)
	team.Id = gotTeam.Id
}

func TestTeamRepo_Create_DuplicateName(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	user1 := &entity.User{
		Username:     "captain1",
		Email:        "captain1@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user1)
	require.NoError(t, err)
	gotUser1, err := userRepo.GetByEmail(ctx, user1.Email)
	require.NoError(t, err)
	user1.Id = gotUser1.Id

	user2 := &entity.User{
		Username:     "captain2",
		Email:        "captain2@example.com",
		PasswordHash: "hash456",
	}
	err = userRepo.Create(ctx, user2)
	require.NoError(t, err)
	gotUser2, err := userRepo.GetByEmail(ctx, user2.Email)
	require.NoError(t, err)
	user2.Id = gotUser2.Id

	team1 := &entity.Team{
		Name:        "DuplicateName",
		InviteToken: "token1",
		CaptainId:   user1.Id,
	}
	err = teamRepo.Create(ctx, team1)
	require.NoError(t, err)
	gotTeam1, err := teamRepo.GetByName(ctx, "DuplicateName")
	require.NoError(t, err)
	team1.Id = gotTeam1.Id

	team2 := &entity.Team{
		Name:        "DuplicateName",
		InviteToken: "token2",
		CaptainId:   user2.Id,
	}
	err = teamRepo.Create(ctx, team2)
	assert.Error(t, err)
	assert.Equal(t, team1.InviteToken, gotTeam1.InviteToken)
}

func TestTeamRepo_GetByID(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "getbyid",
		Email:        "getbyid@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "GetByID Team",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeamByName, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeamByName.Id

	gotTeam, err := teamRepo.GetByID(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, team.Id, gotTeam.Id)
	assert.Equal(t, team.Name, gotTeam.Name)
	assert.Equal(t, team.InviteToken, gotTeam.InviteToken)
	assert.Equal(t, team.CaptainId, gotTeam.CaptainId)
}

func TestTeamRepo_GetByID_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	nonExistentID := uuid.New().String()
	_, err := repo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

func TestTeamRepo_GetByInviteToken(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "invitetoken",
		Email:        "invitetoken@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "InviteToken Team",
		InviteToken: "unique_token_12345",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)

	gotTeam, err := teamRepo.GetByInviteToken(ctx, team.InviteToken)
	require.NoError(t, err)
	team.Id = gotTeam.Id
	assert.Equal(t, team.InviteToken, gotTeam.InviteToken)
}

func TestTeamRepo_GetByInviteToken_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	_, err := repo.GetByInviteToken(ctx, "nonexistent_token")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

func TestTeamRepo_GetByName(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "getbyname",
		Email:        "getbyname@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "GetByName Team",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)

	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id
	assert.Equal(t, team.Name, gotTeam.Name)
}

func TestTeamRepo_GetByName_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewTeamRepo(testDB.DB)
	ctx := context.Background()

	_, err := repo.GetByName(ctx, "nonexistent_team")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}
