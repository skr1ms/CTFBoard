package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Create Tests

func TestTeamUseCase_Create_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	captainID := uuid.New().String()
	user := &entity.User{
		Id:       captainID,
		Username: "captain",
		Email:    "captain@example.com",
		TeamId:   nil,
	}

	teamRepo.On("GetByName", mock.Anything, "TestTeam").Return(nil, entityError.ErrTeamNotFound)
	userRepo.On("GetByID", mock.Anything, captainID).Return(user, nil)
	var teamID string
	teamRepo.On("Create", mock.Anything, mock.MatchedBy(func(t *entity.Team) bool {
		return t.Name == "TestTeam" && t.CaptainId == captainID
	})).Return(nil).Run(func(args mock.Arguments) {
		team := args.Get(1).(*entity.Team)
		teamID = uuid.New().String()
		team.Id = teamID
	})
	teamIDPtr := &teamID
	userRepo.On("UpdateTeamId", mock.Anything, captainID, teamIDPtr).Return(nil)

	uc := NewTeamUseCase(teamRepo, userRepo)

	team, err := uc.Create(context.Background(), "TestTeam", captainID)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, "TestTeam", team.Name)
	assert.Equal(t, captainID, team.CaptainId)
	assert.NotEmpty(t, team.InviteToken)
}

func TestTeamUseCase_Create_TeamNameExists_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	captainID := uuid.New().String()
	existingTeam := &entity.Team{
		Id:   uuid.New().String(),
		Name: "TestTeam",
	}

	teamRepo.On("GetByName", mock.Anything, "TestTeam").Return(existingTeam, nil)

	uc := NewTeamUseCase(teamRepo, userRepo)

	team, err := uc.Create(context.Background(), "TestTeam", captainID)

	assert.Error(t, err)
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_UserAlreadyInTeam_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	captainID := uuid.New().String()
	teamID := uuid.New().String()
	user := &entity.User{
		Id:     captainID,
		TeamId: &teamID,
	}

	teamRepo.On("GetByName", mock.Anything, "TestTeam").Return(nil, entityError.ErrTeamNotFound)
	userRepo.On("GetByID", mock.Anything, captainID).Return(user, nil)

	uc := NewTeamUseCase(teamRepo, userRepo)

	team, err := uc.Create(context.Background(), "TestTeam", captainID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserAlreadyInTeam))
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_GetByNameUnexpected_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	captainID := uuid.New().String()
	expectedError := assert.AnError

	teamRepo.On("GetByName", mock.Anything, "TestTeam").Return(nil, expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	team, err := uc.Create(context.Background(), "TestTeam", captainID)

	assert.Error(t, err)
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_GetByID_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	captainID := uuid.New().String()
	expectedError := assert.AnError

	teamRepo.On("GetByName", mock.Anything, "TestTeam").Return(nil, entityError.ErrTeamNotFound)
	userRepo.On("GetByID", mock.Anything, captainID).Return(nil, expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	team, err := uc.Create(context.Background(), "TestTeam", captainID)

	assert.Error(t, err)
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_Create_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	captainID := uuid.New().String()
	user := &entity.User{
		Id:     captainID,
		TeamId: nil,
	}
	expectedError := assert.AnError

	teamRepo.On("GetByName", mock.Anything, "TestTeam").Return(nil, entityError.ErrTeamNotFound)
	userRepo.On("GetByID", mock.Anything, captainID).Return(user, nil)
	teamRepo.On("Create", mock.Anything, mock.Anything).Return(expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	team, err := uc.Create(context.Background(), "TestTeam", captainID)

	assert.Error(t, err)
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_UpdateTeamId_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	captainID := uuid.New().String()
	user := &entity.User{
		Id:     captainID,
		TeamId: nil,
	}
	expectedError := assert.AnError

	var teamID string
	teamRepo.On("GetByName", mock.Anything, "TestTeam").Return(nil, entityError.ErrTeamNotFound)
	userRepo.On("GetByID", mock.Anything, captainID).Return(user, nil)
	teamRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		team := args.Get(1).(*entity.Team)
		teamID = uuid.New().String()
		team.Id = teamID
	})
	teamIDPtr := &teamID
	userRepo.On("UpdateTeamId", mock.Anything, captainID, teamIDPtr).Return(expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	team, err := uc.Create(context.Background(), "TestTeam", captainID)

	assert.Error(t, err)
	assert.Nil(t, team)
}

// Join Tests

func TestTeamUseCase_Join_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	inviteToken := "test_token_12345"
	userID := uuid.New().String()
	teamID := uuid.New().String()

	team := &entity.Team{
		Id:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}

	user := &entity.User{
		Id:     userID,
		TeamId: nil,
	}

	teamRepo.On("GetByInviteToken", mock.Anything, inviteToken).Return(team, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	teamIDPtr := &teamID
	userRepo.On("UpdateTeamId", mock.Anything, userID, teamIDPtr).Return(nil)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, err := uc.Join(context.Background(), inviteToken, userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, teamID, result.Id)
}

func TestTeamUseCase_Join_UserAlreadyInTeam_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	inviteToken := "test_token_12345"
	userID := uuid.New().String()
	teamID := uuid.New().String()
	existingTeamID := uuid.New().String()

	team := &entity.Team{
		Id:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}

	user := &entity.User{
		Id:     userID,
		TeamId: &existingTeamID,
	}

	otherUser := &entity.User{
		Id:     uuid.New().String(),
		TeamId: &existingTeamID,
	}

	teamRepo.On("GetByInviteToken", mock.Anything, inviteToken).Return(team, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	userRepo.On("GetByTeamId", mock.Anything, existingTeamID).Return([]*entity.User{user, otherUser}, nil)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, err := uc.Join(context.Background(), inviteToken, userID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserAlreadyInTeam))
	assert.Nil(t, result)
}

func TestTeamUseCase_Join_GetByInviteToken_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	inviteToken := "test_token_12345"
	userID := uuid.New().String()
	expectedError := assert.AnError

	teamRepo.On("GetByInviteToken", mock.Anything, inviteToken).Return(nil, expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, err := uc.Join(context.Background(), inviteToken, userID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTeamUseCase_Join_GetByID_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	inviteToken := "test_token_12345"
	userID := uuid.New().String()
	teamID := uuid.New().String()

	team := &entity.Team{
		Id:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}
	expectedError := assert.AnError

	teamRepo.On("GetByInviteToken", mock.Anything, inviteToken).Return(team, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(nil, expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, err := uc.Join(context.Background(), inviteToken, userID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTeamUseCase_Join_UpdateTeamId_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	inviteToken := "test_token_12345"
	userID := uuid.New().String()
	teamID := uuid.New().String()

	team := &entity.Team{
		Id:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}

	user := &entity.User{
		Id:     userID,
		TeamId: nil,
	}
	expectedError := assert.AnError

	teamRepo.On("GetByInviteToken", mock.Anything, inviteToken).Return(team, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	teamIDPtr := &teamID
	userRepo.On("UpdateTeamId", mock.Anything, userID, teamIDPtr).Return(expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, err := uc.Join(context.Background(), inviteToken, userID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// GetByID Tests

func TestTeamUseCase_GetByID_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	teamID := uuid.New().String()
	expectedTeam := &entity.Team{
		Id:          teamID,
		Name:        "TestTeam",
		InviteToken: "token123",
		CaptainId:   uuid.New().String(),
	}

	teamRepo.On("GetByID", mock.Anything, teamID).Return(expectedTeam, nil)

	uc := NewTeamUseCase(teamRepo, userRepo)

	team, err := uc.GetByID(context.Background(), teamID)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, expectedTeam.Id, team.Id)
	assert.Equal(t, expectedTeam.Name, team.Name)
}

func TestTeamUseCase_GetByID_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	teamID := uuid.New().String()
	expectedError := assert.AnError

	teamRepo.On("GetByID", mock.Anything, teamID).Return(nil, expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	team, err := uc.GetByID(context.Background(), teamID)

	assert.Error(t, err)
	assert.Nil(t, team)
}

// GetMyTeam Tests

func TestTeamUseCase_GetMyTeam_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	userID := uuid.New().String()
	teamID := uuid.New().String()

	user := &entity.User{
		Id:     userID,
		TeamId: &teamID,
	}

	team := &entity.Team{
		Id:          teamID,
		Name:        "MyTeam",
		InviteToken: "token123",
		CaptainId:   userID,
	}

	members := []*entity.User{user}

	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	userRepo.On("GetByTeamId", mock.Anything, teamID).Return(members, nil)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, gotMembers, err := uc.GetMyTeam(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, teamID, result.Id)
	assert.Equal(t, "MyTeam", result.Name)
	assert.NotNil(t, gotMembers)
	assert.Equal(t, 1, len(gotMembers))
}

func TestTeamUseCase_GetMyTeam_NoTeam_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	userID := uuid.New().String()

	user := &entity.User{
		Id:     userID,
		TeamId: nil,
	}

	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, members, err := uc.GetMyTeam(context.Background(), userID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
	assert.Nil(t, result)
	assert.Nil(t, members)
}

func TestTeamUseCase_GetMyTeam_GetByIDUser_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	userID := uuid.New().String()
	expectedError := assert.AnError

	userRepo.On("GetByID", mock.Anything, userID).Return(nil, expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, members, err := uc.GetMyTeam(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Nil(t, members)
}

func TestTeamUseCase_GetMyTeam_GetByIDTeam_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	userID := uuid.New().String()
	teamID := uuid.New().String()

	user := &entity.User{
		Id:     userID,
		TeamId: &teamID,
	}
	expectedError := assert.AnError

	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	teamRepo.On("GetByID", mock.Anything, teamID).Return(nil, expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, members, err := uc.GetMyTeam(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Nil(t, members)
}

// GetTeamMembers Tests

func TestTeamUseCase_GetTeamMembers_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	teamID := uuid.New().String()
	members := []*entity.User{
		{
			Id:       uuid.New().String(),
			Username: "member1",
			TeamId:   &teamID,
		},
		{
			Id:       uuid.New().String(),
			Username: "member2",
			TeamId:   &teamID,
		},
	}

	userRepo.On("GetByTeamId", mock.Anything, teamID).Return(members, nil)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, err := uc.GetTeamMembers(context.Background(), teamID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
}

func TestTeamUseCase_GetTeamMembers_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)

	teamID := uuid.New().String()
	expectedError := assert.AnError

	userRepo.On("GetByTeamId", mock.Anything, teamID).Return(nil, expectedError)

	uc := NewTeamUseCase(teamRepo, userRepo)

	result, err := uc.GetTeamMembers(context.Background(), teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}
