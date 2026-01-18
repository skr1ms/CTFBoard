package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create Tests

func TestTeamRepo_Create(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "captain")

	assert.NotEmpty(t, team.Id)
	gotTeam, err := f.TeamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	assert.NotZero(t, gotTeam.CreatedAt)
}

func TestTeamRepo_Create_DuplicateName(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team1 := f.CreateUserWithTeam(t, "duplicate_1")

	user2 := f.CreateUser(t, "duplicate_2")

	team2 := &entity.Team{
		Name:        team1.Name,
		InviteToken: "token2",
		CaptainId:   user2.Id,
	}
	err := f.TeamRepo.Create(ctx, team2)
	assert.Error(t, err)

	gotTeam1, err := f.TeamRepo.GetByName(ctx, team1.Name)
	require.NoError(t, err)
	assert.Equal(t, team1.InviteToken, gotTeam1.InviteToken)
}

// GetByID Tests

func TestTeamRepo_GetByID(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_by_id")

	gotTeam, err := f.TeamRepo.GetByID(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, team.Id, gotTeam.Id)
	assert.Equal(t, team.Name, gotTeam.Name)
	assert.Equal(t, team.InviteToken, gotTeam.InviteToken)
	assert.Equal(t, team.CaptainId, gotTeam.CaptainId)
}

func TestTeamRepo_GetByID_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	nonExistentID := uuid.New().String()
	_, err := f.TeamRepo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// GetByInviteToken Tests

func TestTeamRepo_GetByInviteToken(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "invite_token")

	gotTeam, err := f.TeamRepo.GetByInviteToken(ctx, team.InviteToken)
	require.NoError(t, err)
	assert.Equal(t, team.Id, gotTeam.Id)
	assert.Equal(t, team.InviteToken, gotTeam.InviteToken)
}

func TestTeamRepo_GetByInviteToken_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, err := f.TeamRepo.GetByInviteToken(ctx, "nonexistent_token")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// GetByName Tests

func TestTeamRepo_GetByName(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_by_name")

	gotTeam, err := f.TeamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	assert.Equal(t, team.Id, gotTeam.Id)
	assert.Equal(t, team.Name, gotTeam.Name)
}

func TestTeamRepo_GetByName_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, err := f.TeamRepo.GetByName(ctx, "nonexistent_team")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}
