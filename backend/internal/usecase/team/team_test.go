package team

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTeamUseCase_Create_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	captainID := uuid.New()
	user := h.NewUser(captainID, nil, "captain", "captain@example.com")

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	deps.txRepo.EXPECT().GetTeamByNameTx(mock.Anything, mock.Anything, "TestTeam").Return(nil, entityError.ErrTeamNotFound).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	deps.txRepo.EXPECT().CreateTeamTx(mock.Anything, mock.Anything, mock.MatchedBy(func(t *entity.Team) bool {
		return t.Name == "TestTeam" && t.CaptainID == captainID
	})).Return(nil).Run(func(_ context.Context, _ pgx.Tx, team *entity.Team) {
		team.ID = uuid.New()
		team.InviteToken = uuid.New()
	}).Once()
	deps.txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, captainID, mock.Anything).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	uc := h.CreateUseCase()

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, "TestTeam", team.Name)
	assert.Equal(t, captainID, team.CaptainID)
	assert.NotEmpty(t, team.InviteToken)
}

func TestTeamUseCase_Create_WithSoloTeam_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	captainID := uuid.New()
	oldTeamID := uuid.New()
	user := h.NewUser(captainID, &oldTeamID, "captain", "captain@example.com")

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	deps.txRepo.EXPECT().GetTeamByNameTx(mock.Anything, mock.Anything, "NewTeam").Return(nil, entityError.ErrTeamNotFound).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	deps.txRepo.EXPECT().GetTeamByIDTx(mock.Anything, mock.Anything, oldTeamID).Return(&entity.Team{ID: oldTeamID, IsSolo: true}, nil).Once()
	deps.txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, oldTeamID).Return([]*entity.User{user}, nil).Once()
	deps.txRepo.EXPECT().SoftDeleteTeamTx(mock.Anything, mock.Anything, oldTeamID).Return(nil).Once()
	deps.txRepo.EXPECT().DeleteSolvesByTeamIDTx(mock.Anything, mock.Anything, oldTeamID).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionDeleted
	})).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamTx(mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, _ pgx.Tx, team *entity.Team) {
		team.ID = uuid.New()
		team.InviteToken = uuid.New()
	}).Once()
	deps.txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, captainID, mock.Anything).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionCreated
	})).Return(nil).Once()

	uc := h.CreateUseCase()

	team, err := uc.Create(context.Background(), "NewTeam", captainID, false, true)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, "NewTeam", team.Name)
}

func TestTeamUseCase_Create_TeamNameExists_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	captainID := uuid.New()
	existingTeam := &entity.Team{
		ID:   uuid.New(),
		Name: "TestTeam",
	}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	deps.txRepo.EXPECT().GetTeamByNameTx(mock.Anything, mock.Anything, "TestTeam").Return(existingTeam, nil).Once()

	uc := h.CreateUseCase()

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.Error(t, err)
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_UserAlreadyInMultiMemberTeam_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	captainID := uuid.New()
	teamID := uuid.New()
	otherUserID := uuid.New()
	user := &entity.User{
		ID:     captainID,
		TeamID: &teamID,
	}
	otherUser := &entity.User{
		ID:     otherUserID,
		TeamID: &teamID,
	}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once() // Add compRepo expectation
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	deps.txRepo.EXPECT().GetTeamByNameTx(mock.Anything, mock.Anything, "TestTeam").Return(nil, entityError.ErrTeamNotFound).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	deps.txRepo.EXPECT().GetTeamByIDTx(mock.Anything, mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsSolo: false}, nil).Once()
	deps.txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return([]*entity.User{user, otherUser}, nil).Once()

	uc := h.CreateUseCase()

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserAlreadyInTeam))
	assert.Nil(t, team)
}

func TestTeamUseCase_Join_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	inviteToken := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	team := &entity.Team{
		ID:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}

	user := &entity.User{
		ID:     userID,
		TeamID: nil,
	}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	deps.txRepo.EXPECT().GetTeamByInviteTokenTx(mock.Anything, mock.Anything, inviteToken).Return(team, nil).Once()
	deps.txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	deps.txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return([]*entity.User{}, nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	deps.txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, userID, &teamID).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionJoined
	})).Return(nil).Once()

	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := h.CreateUseCase()

	result, err := uc.Join(context.Background(), inviteToken, userID, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, teamID, result.ID)
}

func TestTeamUseCase_Join_TeamFull_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	inviteToken := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	team := &entity.Team{
		ID:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}

	existingMembers := make([]*entity.User, 10)
	for i := 0; i < 10; i++ {
		existingMembers[i] = &entity.User{ID: uuid.New()}
	}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	deps.txRepo.EXPECT().GetTeamByInviteTokenTx(mock.Anything, mock.Anything, inviteToken).Return(team, nil).Once()
	deps.txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	deps.txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return(existingMembers, nil).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := h.CreateUseCase()

	result, err := uc.Join(context.Background(), inviteToken, userID, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamFull))
	assert.Nil(t, result)
}

func TestTeamUseCase_Join_WithSoloTeam_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	inviteToken := uuid.New()
	userID := uuid.New()
	newTeamID := uuid.New()
	oldTeamID := uuid.New()

	newTeam := &entity.Team{
		ID:          newTeamID,
		Name:        "NewTeam",
		InviteToken: inviteToken,
	}

	user := &entity.User{
		ID:     userID,
		TeamID: &oldTeamID,
	}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	deps.txRepo.EXPECT().GetTeamByInviteTokenTx(mock.Anything, mock.Anything, inviteToken).Return(newTeam, nil).Once()
	deps.txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, newTeamID).Return(nil).Once()
	deps.txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, newTeamID).Return([]*entity.User{}, nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	deps.txRepo.EXPECT().GetTeamByIDTx(mock.Anything, mock.Anything, oldTeamID).Return(&entity.Team{ID: oldTeamID, IsSolo: true}, nil).Once()
	deps.txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, oldTeamID).Return([]*entity.User{user}, nil).Once()
	deps.txRepo.EXPECT().SoftDeleteTeamTx(mock.Anything, mock.Anything, oldTeamID).Return(nil).Once()
	deps.txRepo.EXPECT().DeleteSolvesByTeamIDTx(mock.Anything, mock.Anything, oldTeamID).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionDeleted
	})).Return(nil).Once()
	deps.txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, userID, &newTeamID).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionJoined
	})).Return(nil).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := h.CreateUseCase()

	result, err := uc.Join(context.Background(), inviteToken, userID, true)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newTeamID, result.ID)
}

func TestTeamUseCase_Leave_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	captainID := uuid.New()
	teamID := uuid.New()

	user := &entity.User{
		ID:     userID,
		TeamID: &teamID,
	}

	team := &entity.Team{
		ID:        teamID,
		CaptainID: captainID,
	}

	members := []*entity.User{user, {ID: captainID, TeamID: &teamID}}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	deps.txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	deps.txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return(members, nil).Once()
	deps.txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, userID, (*uuid.UUID)(nil)).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionLeft
	})).Return(nil).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := h.CreateUseCase()

	err := uc.Leave(context.Background(), userID)

	assert.NoError(t, err)
}

func TestTeamUseCase_Leave_CaptainCannotLeave_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	captainID := uuid.New()
	teamID := uuid.New()

	captain := &entity.User{
		ID:     captainID,
		TeamID: &teamID,
	}

	team := &entity.Team{
		ID:        teamID,
		CaptainID: captainID,
	}

	members := []*entity.User{captain, {ID: uuid.New(), TeamID: &teamID}}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	deps.txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	deps.txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return(members, nil).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := h.CreateUseCase()

	err := uc.Leave(context.Background(), captainID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrNotCaptain))
}

func TestTeamUseCase_TransferCaptain_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	captainID := uuid.New()
	newCaptainID := uuid.New()
	teamID := uuid.New()

	captain := &entity.User{
		ID:     captainID,
		TeamID: &teamID,
	}

	newCaptain := &entity.User{
		ID:     newCaptainID,
		TeamID: &teamID,
	}

	team := &entity.Team{
		ID:        teamID,
		CaptainID: captainID,
	}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	deps.txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, newCaptainID).Return(newCaptain, nil).Once()
	deps.txRepo.EXPECT().UpdateTeamCaptainTx(mock.Anything, mock.Anything, teamID, newCaptainID).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionCaptainTransfer
	})).Return(nil).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := h.CreateUseCase()

	err := uc.TransferCaptain(context.Background(), captainID, newCaptainID)

	assert.NoError(t, err)
}

func TestTeamUseCase_TransferCaptain_NotCaptain_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	realCaptainID := uuid.New()
	newCaptainID := uuid.New()
	teamID := uuid.New()

	user := &entity.User{
		ID:     userID,
		TeamID: &teamID,
	}

	team := &entity.Team{
		ID:        teamID,
		CaptainID: realCaptainID,
	}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	deps.txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()

	uc := h.CreateUseCase()

	err := uc.TransferCaptain(context.Background(), userID, newCaptainID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrNotCaptain))
}

func TestTeamUseCase_GetByID_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	teamID := uuid.New()
	expectedTeam := &entity.Team{
		ID:          teamID,
		Name:        "TestTeam",
		InviteToken: uuid.New(),
		CaptainID:   uuid.New(),
	}

	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(expectedTeam, nil).Once()

	uc := h.CreateUseCase()

	team, err := uc.GetByID(context.Background(), teamID)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, expectedTeam.ID, team.ID)
	assert.Equal(t, expectedTeam.Name, team.Name)
}

func TestTeamUseCase_GetMyTeam_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	teamID := uuid.New()

	user := &entity.User{
		ID:     userID,
		TeamID: &teamID,
	}

	team := &entity.Team{
		ID:          teamID,
		Name:        "MyTeam",
		InviteToken: uuid.New(),
		CaptainID:   userID,
	}

	members := []*entity.User{user}

	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	deps.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Once()

	uc := h.CreateUseCase()

	result, gotMembers, err := uc.GetMyTeam(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, teamID, result.ID)
	assert.Equal(t, "MyTeam", result.Name)
	assert.NotNil(t, gotMembers)
	assert.Equal(t, 1, len(gotMembers))
}

func TestTeamUseCase_GetTeamMembers_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	teamID := uuid.New()
	members := []*entity.User{
		{
			ID:       uuid.New(),
			Username: "member1",
			TeamID:   &teamID,
		},
		{
			ID:       uuid.New(),
			Username: "member2",
			TeamID:   &teamID,
		},
	}

	deps.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Once()

	uc := h.CreateUseCase()

	result, err := uc.GetTeamMembers(context.Background(), teamID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
}

func TestTeamUseCase_CreateSoloTeam_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	user := &entity.User{ID: userID, Username: "solo_user"}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once() // Add compRepo expectation
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()

	deps.txRepo.EXPECT().GetTeamByNameTx(mock.Anything, mock.Anything, "solo_user").Return(nil, entityError.ErrTeamNotFound).Once()

	deps.txRepo.EXPECT().CreateTeamTx(mock.Anything, mock.Anything, mock.MatchedBy(func(tm *entity.Team) bool {
		return tm.IsSolo == true && tm.CaptainID == userID && tm.Name == "solo_user"
	})).Return(nil).Run(func(ctx context.Context, tx pgx.Tx, tm *entity.Team) {
		tm.ID = uuid.New()
	}).Once()

	deps.txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, userID, mock.Anything).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionCreated
	})).Return(nil).Once()

	uc := h.CreateUseCase()
	team, err := uc.CreateSoloTeam(context.Background(), userID, false)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.True(t, team.IsSolo)
	assert.Equal(t, "solo_user", team.Name)
}

func TestTeamUseCase_CreateSoloTeam_Error_AlreadyInTeam(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	teamID := uuid.New()
	user := &entity.User{ID: userID, TeamID: &teamID}

	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()
	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, userID).Return(nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	deps.txRepo.EXPECT().GetTeamByIDTx(mock.Anything, mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsSolo: false, IsAutoCreated: false}, nil).Once()
	deps.txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return([]*entity.User{user}, nil).Once()

	uc := h.CreateUseCase()
	team, err := uc.CreateSoloTeam(context.Background(), userID, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserAlreadyInTeam))
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_Error_RosterFrozen(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := h.CreateUseCase()
	team, err := uc.Create(context.Background(), "test_team", uuid.New(), false, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
	assert.Nil(t, team)
}

func TestTeamUseCase_Join_Error_RosterFrozen(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := h.CreateUseCase()
	team, err := uc.Join(context.Background(), uuid.New(), uuid.New(), false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
	assert.Nil(t, team)
}

func TestTeamUseCase_CreateSoloTeam_Error_RosterFrozen(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := h.CreateUseCase()
	team, err := uc.CreateSoloTeam(context.Background(), uuid.New(), false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
	assert.Nil(t, team)
}

func TestTeamUseCase_Leave_Error_RosterFrozen(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := h.CreateUseCase()
	err := uc.Leave(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
}

func TestTeamUseCase_TransferCaptain_Error_RosterFrozen(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := h.CreateUseCase()
	err := uc.TransferCaptain(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
}

func TestTeamUseCase_DisbandTeam_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	captainID := uuid.New()
	teamID := uuid.New()
	captain := &entity.User{ID: captainID, TeamID: &teamID}

	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: true}, nil).Once()
	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()

	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	deps.txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, CaptainID: captainID, Name: "test_team"}, nil).Once()

	deps.txRepo.EXPECT().SoftDeleteTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	deps.txRepo.EXPECT().GetUsersByTeamIDTx(mock.Anything, mock.Anything, teamID).Return([]*entity.User{captain}, nil).Once()
	deps.txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, captainID, (*uuid.UUID)(nil)).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		return l.Action == entity.TeamActionDeleted && l.TeamID == teamID && l.UserID == captainID
	})).Return(nil).Once()

	uc := h.CreateUseCase()
	err := uc.DisbandTeam(context.Background(), captainID)

	assert.NoError(t, err)
}

func TestTeamUseCase_DisbandTeam_Error_RosterFrozen(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := h.CreateUseCase()
	err := uc.DisbandTeam(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
}

func TestTeamUseCase_KickMember_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	captainID := uuid.New()
	targetID := uuid.New()
	teamID := uuid.New()
	captain := &entity.User{ID: captainID, TeamID: &teamID}
	target := &entity.User{ID: targetID, TeamID: &teamID}

	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: true}, nil).Once()
	deps.txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return fn(ctx, nil)
	}).Once()

	deps.txRepo.EXPECT().LockUserTx(mock.Anything, mock.Anything, captainID).Return(nil).Once()
	deps.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()

	deps.txRepo.EXPECT().LockTeamTx(mock.Anything, mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, CaptainID: captainID, Name: "test_team"}, nil).Once()

	deps.userRepo.EXPECT().GetByID(mock.Anything, targetID).Return(target, nil).Once()

	deps.txRepo.EXPECT().UpdateUserTeamIDTx(mock.Anything, mock.Anything, targetID, (*uuid.UUID)(nil)).Return(nil).Once()
	deps.txRepo.EXPECT().CreateTeamAuditLogTx(mock.Anything, mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		targetIDStr := targetID.String()
		detailsTargetID, ok := l.Details["target_user_id"].(string)
		return l.Action == entity.TeamActionMemberKicked &&
			l.TeamID == teamID &&
			l.UserID == captainID &&
			ok && detailsTargetID == targetIDStr
	})).Return(nil).Once()

	uc := h.CreateUseCase()
	err := uc.KickMember(context.Background(), captainID, targetID)

	assert.NoError(t, err)
}

func TestTeamUseCase_KickMember_Error_RosterFrozen(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()

	uc := h.CreateUseCase()
	err := uc.KickMember(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
}

func TestTeamUseCase_GetByID_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	teamID := uuid.New()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, entityError.ErrTeamNotFound).Once()

	uc := h.CreateUseCase()

	team, err := uc.GetByID(context.Background(), teamID)

	assert.Error(t, err)
	assert.Nil(t, team)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

func TestTeamUseCase_GetMyTeam_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, entityError.ErrUserNotFound).Once()

	uc := h.CreateUseCase()

	team, members, err := uc.GetMyTeam(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, team)
	assert.Nil(t, members)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

func TestTeamUseCase_GetTeamMembers_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	teamID := uuid.New()
	deps.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(nil, errors.New("db error")).Once()

	uc := h.CreateUseCase()

	members, err := uc.GetTeamMembers(context.Background(), teamID)

	assert.Error(t, err)
	assert.Nil(t, members)
}

func TestTeamUseCase_BanTeam_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	teamID := uuid.New()
	team := &entity.Team{ID: teamID, Name: "Team"}
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	deps.teamRepo.EXPECT().Ban(mock.Anything, teamID, "reason").Return(nil).Once()

	uc := h.CreateUseCase()

	err := uc.BanTeam(context.Background(), teamID, "reason")

	assert.NoError(t, err)
}

func TestTeamUseCase_BanTeam_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	teamID := uuid.New()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, entityError.ErrTeamNotFound).Once()

	uc := h.CreateUseCase()

	err := uc.BanTeam(context.Background(), teamID, "reason")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

func TestTeamUseCase_UnbanTeam_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	teamID := uuid.New()
	team := &entity.Team{ID: teamID, Name: "Team"}
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	deps.teamRepo.EXPECT().Unban(mock.Anything, teamID).Return(nil).Once()

	uc := h.CreateUseCase()

	err := uc.UnbanTeam(context.Background(), teamID)

	assert.NoError(t, err)
}

func TestTeamUseCase_UnbanTeam_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	teamID := uuid.New()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, errors.New("db error")).Once()

	uc := h.CreateUseCase()

	err := uc.UnbanTeam(context.Background(), teamID)

	assert.Error(t, err)
}

func TestTeamUseCase_SetHidden_Success(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	teamID := uuid.New()
	team := &entity.Team{ID: teamID, Name: "Team"}
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	deps.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, true).Return(nil).Once()

	uc := h.CreateUseCase()

	err := uc.SetHidden(context.Background(), teamID, true)

	assert.NoError(t, err)
}

func TestTeamUseCase_SetHidden_Error(t *testing.T) {
	h := NewTeamTestHelper(t)
	deps := h.Deps()

	teamID := uuid.New()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, entityError.ErrTeamNotFound).Once()

	uc := h.CreateUseCase()

	err := uc.SetHidden(context.Background(), teamID, true)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}
