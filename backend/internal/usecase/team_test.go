package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTeamUseCase_Create_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	captainID := uuid.New()
	user := &entity.User{
		Id:       captainID,
		Username: "captain",
		Email:    "captain@example.com",
		TeamId:   nil,
	}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	txRepo.EXPECT().GetTeamByNameTx(mock.Anything, mock.Anything, "TestTeam").Return(nil, entityError.ErrTeamNotFound).Once()
	userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	txRepo.EXPECT().CreateTeamTx(mock.Anything, mock.Anything, mock.MatchedBy(func(t *entity.Team) bool {
		return t.Name == "TestTeam" && t.CaptainId == captainID
	})).Return(nil).Run(func(ctx context.Context, tx pgx.Tx, team *entity.Team) {
		team.Id = uuid.New()
	}).Once()
	txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, captainID, mock.Anything).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, "TestTeam", team.Name)
	assert.Equal(t, captainID, team.CaptainId)
	assert.NotEmpty(t, team.InviteToken)
}

func TestTeamUseCase_Create_WithSoloTeam_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	captainID := uuid.New()
	oldTeamID := uuid.New()
	user := &entity.User{
		Id:       captainID,
		Username: "captain",
		Email:    "captain@example.com",
		TeamId:   &oldTeamID,
	}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	txRepo.EXPECT().GetTeamByNameTx(mock.Anything, mock.Anything, "NewTeam").Return(nil, entityError.ErrTeamNotFound).Once()
	userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	txRepo.EXPECT().GetTeamByIDTx(mock.Anything, mock.Anything, oldTeamID).Return(&entity.Team{Id: oldTeamID, IsSolo: true}, nil).Once()
	txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, oldTeamID).Return([]*entity.User{user}, nil).Once()
	txRepo.EXPECT().SoftDeleteTeamTx(mock.Anything, mock.Anything, oldTeamID).Return(nil).Once()
	txRepo.EXPECT().DeleteSolvesByTeamIDTx(mock.Anything, mock.Anything, oldTeamID).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionDeleted
	})).Return(nil).Once()
	txRepo.EXPECT().CreateTeamTx(mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(ctx context.Context, tx pgx.Tx, team *entity.Team) {
		team.Id = uuid.New()
	}).Once()
	txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, captainID, mock.Anything).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionCreated
	})).Return(nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	team, err := uc.Create(context.Background(), "NewTeam", captainID, false, true)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, "NewTeam", team.Name)
}

func TestTeamUseCase_Create_TeamNameExists_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	captainID := uuid.New()
	existingTeam := &entity.Team{
		Id:   uuid.New(),
		Name: "TestTeam",
	}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	txRepo.EXPECT().GetTeamByNameTx(mock.Anything, mock.Anything, "TestTeam").Return(existingTeam, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.Error(t, err)
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_UserAlreadyInMultiMemberTeam_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	captainID := uuid.New()
	teamID := uuid.New()
	otherUserID := uuid.New()
	user := &entity.User{
		Id:     captainID,
		TeamId: &teamID,
	}
	otherUser := &entity.User{
		Id:     otherUserID,
		TeamId: &teamID,
	}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once() // Add compRepo expectation
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	txRepo.EXPECT().GetTeamByNameTx(mock.Anything, mock.Anything, "TestTeam").Return(nil, entityError.ErrTeamNotFound).Once()
	userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	txRepo.EXPECT().GetTeamByIDTx(mock.Anything, mock.Anything, teamID).Return(&entity.Team{Id: teamID, IsSolo: false}, nil).Once()
	txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return([]*entity.User{user, otherUser}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserAlreadyInTeam))
	assert.Nil(t, team)
}

func TestTeamUseCase_Join_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	inviteToken := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	team := &entity.Team{
		Id:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}

	user := &entity.User{
		Id:     userID,
		TeamId: nil,
	}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	txRepo.EXPECT().GetTeamByInviteTokenTx(mock.Anything, mock.Anything, inviteToken).Return(team, nil).Once()
	txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return([]*entity.User{}, nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, userID, &teamID).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionJoined
	})).Return(nil).Once()

	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	result, err := uc.Join(context.Background(), inviteToken, userID, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, teamID, result.Id)
}

func TestTeamUseCase_Join_TeamFull_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	inviteToken := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	team := &entity.Team{
		Id:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}

	existingMembers := make([]*entity.User, 10)
	for i := 0; i < 10; i++ {
		existingMembers[i] = &entity.User{Id: uuid.New()}
	}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	txRepo.EXPECT().GetTeamByInviteTokenTx(mock.Anything, mock.Anything, inviteToken).Return(team, nil).Once()
	txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return(existingMembers, nil).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	result, err := uc.Join(context.Background(), inviteToken, userID, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamFull))
	assert.Nil(t, result)
}

func TestTeamUseCase_Join_WithSoloTeam_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	inviteToken := uuid.New()
	userID := uuid.New()
	newTeamID := uuid.New()
	oldTeamID := uuid.New()

	newTeam := &entity.Team{
		Id:          newTeamID,
		Name:        "NewTeam",
		InviteToken: inviteToken,
	}

	user := &entity.User{
		Id:     userID,
		TeamId: &oldTeamID,
	}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	txRepo.EXPECT().GetTeamByInviteTokenTx(mock.Anything, mock.Anything, inviteToken).Return(newTeam, nil).Once()
	txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, newTeamID).Return(nil).Once()
	txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, newTeamID).Return([]*entity.User{}, nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	txRepo.EXPECT().GetTeamByIDTx(mock.Anything, mock.Anything, oldTeamID).Return(&entity.Team{Id: oldTeamID, IsSolo: true}, nil).Once()
	txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, oldTeamID).Return([]*entity.User{user}, nil).Once()
	txRepo.EXPECT().SoftDeleteTeamTx(mock.Anything, mock.Anything, oldTeamID).Return(nil).Once()
	txRepo.EXPECT().DeleteSolvesByTeamIDTx(mock.Anything, mock.Anything, oldTeamID).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionDeleted
	})).Return(nil).Once()
	txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, userID, &newTeamID).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionJoined
	})).Return(nil).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	result, err := uc.Join(context.Background(), inviteToken, userID, true)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newTeamID, result.Id)
}

func TestTeamUseCase_Leave_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	userID := uuid.New()
	captainID := uuid.New()
	teamID := uuid.New()

	user := &entity.User{
		Id:     userID,
		TeamId: &teamID,
	}

	team := &entity.Team{
		Id:        teamID,
		CaptainId: captainID,
	}

	members := []*entity.User{user, {Id: captainID, TeamId: &teamID}}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return(members, nil).Once()
	txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, userID, (*uuid.UUID)(nil)).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionLeft
	})).Return(nil).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	err := uc.Leave(context.Background(), userID)

	assert.NoError(t, err)
}

func TestTeamUseCase_Leave_CaptainCannotLeave_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	captainID := uuid.New()
	teamID := uuid.New()

	captain := &entity.User{
		Id:     captainID,
		TeamId: &teamID,
	}

	team := &entity.Team{
		Id:        teamID,
		CaptainId: captainID,
	}

	members := []*entity.User{captain, {Id: uuid.New(), TeamId: &teamID}}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return(members, nil).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	err := uc.Leave(context.Background(), captainID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrNotCaptain))
}

func TestTeamUseCase_TransferCaptain_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	captainID := uuid.New()
	newCaptainID := uuid.New()
	teamID := uuid.New()

	captain := &entity.User{
		Id:     captainID,
		TeamId: &teamID,
	}

	newCaptain := &entity.User{
		Id:     newCaptainID,
		TeamId: &teamID,
	}

	team := &entity.Team{
		Id:        teamID,
		CaptainId: captainID,
	}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, newCaptainID).Return(newCaptain, nil).Once()
	txRepo.EXPECT().UpdateTeamCaptainTx(mock.Anything, mock.Anything, teamID, newCaptainID).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionCaptainTransfer
	})).Return(nil).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	err := uc.TransferCaptain(context.Background(), captainID, newCaptainID)

	assert.NoError(t, err)
}

func TestTeamUseCase_TransferCaptain_NotCaptain_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	userID := uuid.New()
	realCaptainID := uuid.New()
	newCaptainID := uuid.New()
	teamID := uuid.New()

	user := &entity.User{
		Id:     userID,
		TeamId: &teamID,
	}

	team := &entity.Team{
		Id:        teamID,
		CaptainId: realCaptainID,
	}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	err := uc.TransferCaptain(context.Background(), userID, newCaptainID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrNotCaptain))
}

func TestTeamUseCase_GetByID_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	teamID := uuid.New()
	expectedTeam := &entity.Team{
		Id:          teamID,
		Name:        "TestTeam",
		InviteToken: uuid.New(),
		CaptainId:   uuid.New(),
	}

	teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(expectedTeam, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	team, err := uc.GetByID(context.Background(), teamID)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, expectedTeam.Id, team.Id)
	assert.Equal(t, expectedTeam.Name, team.Name)
}

func TestTeamUseCase_GetMyTeam_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	userID := uuid.New()
	teamID := uuid.New()

	user := &entity.User{
		Id:     userID,
		TeamId: &teamID,
	}

	team := &entity.Team{
		Id:          teamID,
		Name:        "MyTeam",
		InviteToken: uuid.New(),
		CaptainId:   userID,
	}

	members := []*entity.User{user}

	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	userRepo.EXPECT().GetByTeamId(mock.Anything, teamID).Return(members, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	result, gotMembers, err := uc.GetMyTeam(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, teamID, result.Id)
	assert.Equal(t, "MyTeam", result.Name)
	assert.NotNil(t, gotMembers)
	assert.Equal(t, 1, len(gotMembers))
}

func TestTeamUseCase_GetTeamMembers_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	teamID := uuid.New()
	members := []*entity.User{
		{
			Id:       uuid.New(),
			Username: "member1",
			TeamId:   &teamID,
		},
		{
			Id:       uuid.New(),
			Username: "member2",
			TeamId:   &teamID,
		},
	}

	userRepo.EXPECT().GetByTeamId(mock.Anything, teamID).Return(members, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)

	result, err := uc.GetTeamMembers(context.Background(), teamID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
}

func TestTeamUseCase_CreateSoloTeam_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	userID := uuid.New()
	user := &entity.User{Id: userID, Username: "solo_user"}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once() // Add compRepo expectation
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()

	txRepo.EXPECT().GetTeamByNameTx(mock.Anything, mock.Anything, "solo_user").Return(nil, entityError.ErrTeamNotFound).Once()

	txRepo.EXPECT().CreateTeamTx(mock.Anything, mock.Anything, mock.MatchedBy(func(tm *entity.Team) bool {
		return tm.IsSolo == true && tm.CaptainId == userID && tm.Name == "solo_user"
	})).Return(nil).Run(func(ctx context.Context, tx pgx.Tx, tm *entity.Team) {
		tm.Id = uuid.New()
	}).Once()

	txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, userID, mock.Anything).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionCreated
	})).Return(nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	team, err := uc.CreateSoloTeam(context.Background(), userID, false)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.True(t, team.IsSolo)
	assert.Equal(t, "solo_user", team.Name)
}

func TestTeamUseCase_CreateSoloTeam_Error_AlreadyInTeam(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	userID := uuid.New()
	teamID := uuid.New()
	user := &entity.User{Id: userID, TeamId: &teamID}

	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	txRepo.EXPECT().GetTeamByIDTx(mock.Anything, mock.Anything, teamID).Return(&entity.Team{Id: teamID, IsSolo: false, IsAutoCreated: false}, nil).Once()
	txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return([]*entity.User{user}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	team, err := uc.CreateSoloTeam(context.Background(), userID, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserAlreadyInTeam))
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_Error_RosterFrozen(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	team, err := uc.Create(context.Background(), "test_team", uuid.New(), false, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
	assert.Nil(t, team)
}

func TestTeamUseCase_Join_Error_RosterFrozen(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	team, err := uc.Join(context.Background(), uuid.New(), uuid.New(), false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
	assert.Nil(t, team)
}

func TestTeamUseCase_CreateSoloTeam_Error_RosterFrozen(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	team, err := uc.CreateSoloTeam(context.Background(), uuid.New(), false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
	assert.Nil(t, team)
}

func TestTeamUseCase_Leave_Error_RosterFrozen(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	err := uc.Leave(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
}

func TestTeamUseCase_TransferCaptain_Error_RosterFrozen(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	err := uc.TransferCaptain(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
}

func TestTeamUseCase_DisbandTeam_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	captainID := uuid.New()
	teamID := uuid.New()
	captain := &entity.User{Id: captainID, TeamId: &teamID}

	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: true}, nil).Once()
	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()

	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{Id: teamID, CaptainId: captainID, Name: "test_team"}, nil).Once()

	txRepo.EXPECT().SoftDeleteTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return([]*entity.User{captain}, nil).Once()
	txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, captainID, (*uuid.UUID)(nil)).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		return l.Action == entity.TeamActionDeleted && l.TeamId == teamID && l.UserId == captainID
	})).Return(nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	err := uc.DisbandTeam(context.Background(), captainID)

	assert.NoError(t, err)
}

func TestTeamUseCase_DisbandTeam_Error_RosterFrozen(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	err := uc.DisbandTeam(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
}

func TestTeamUseCase_KickMember_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	captainID := uuid.New()
	targetID := uuid.New()
	teamID := uuid.New()
	captain := &entity.User{Id: captainID, TeamId: &teamID}
	target := &entity.User{Id: targetID, TeamId: &teamID}

	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: true}, nil).Once()
	txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()

	txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()

	txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{Id: teamID, CaptainId: captainID, Name: "test_team"}, nil).Once()

	userRepo.EXPECT().GetByID(mock.Anything, targetID).Return(target, nil).Once()

	txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, targetID, (*uuid.UUID)(nil)).Return(nil).Once()
	txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		targetIDStr := targetID.String()
		detailsTargetID, ok := l.Details["target_user_id"].(string)
		return l.Action == entity.TeamActionMemberKicked &&
			l.TeamId == teamID &&
			l.UserId == captainID &&
			ok && detailsTargetID == targetIDStr
	})).Return(nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	err := uc.KickMember(context.Background(), captainID, targetID)

	assert.NoError(t, err)
}

func TestTeamUseCase_KickMember_Error_RosterFrozen(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)

	compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	err := uc.KickMember(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
}
