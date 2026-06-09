package team

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestTeamUseCase_Join_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	inviteToken := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	team := &domain.Team{
		ID:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}

	user := &domain.User{
		ID:     userID,
		TeamID: nil,
	}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByInviteToken(mock.Anything, inviteToken).Return(team, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{}, nil).Times(2)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, &teamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *domain.TeamAuditLog) bool {
		return log.Action == domain.TeamActionJoined
	})).Return(nil).Once()

	uc := d.createUseCase()

	result, err := uc.Join(context.Background(), inviteToken, userID, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, teamID, result.ID)
}

func TestTeamUseCase_Join_TeamFull_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	inviteToken := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	team := &domain.Team{
		ID:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}

	existingMembers := make([]*domain.User, 10)

	for i := range 10 {
		existingMembers[i] = &domain.User{ID: uuid.New()}
	}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true, MaxTeamSize: 10}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true, MaxTeamSize: 10}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByInviteToken(mock.Anything, inviteToken).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(existingMembers, nil).Once()

	uc := d.createUseCase()

	result, err := uc.Join(context.Background(), inviteToken, userID, false)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamFull)
	assert.Nil(t, result)
}

func TestTeamUseCase_Join_WithSoloTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	inviteToken := uuid.New()
	userID := uuid.New()
	newTeamID := uuid.New()
	oldTeamID := uuid.New()

	newTeam := &domain.Team{
		ID:          newTeamID,
		Name:        "NewTeam",
		InviteToken: inviteToken,
	}

	user := &domain.User{
		ID:     userID,
		TeamID: &oldTeamID,
	}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByInviteToken(mock.Anything, inviteToken).Return(newTeam, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, mock.Anything).Return(nil).Times(4)
	d.teamRepo.EXPECT().GetByID(mock.Anything, newTeamID).Return(newTeam, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, newTeamID).Return([]*domain.User{}, nil).Times(2)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, oldTeamID).Return(&domain.Team{ID: oldTeamID, IsSolo: true}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, oldTeamID).Return([]*domain.User{user}, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, oldTeamID).Return([]*domain.SolveWithDetails{}, nil).Once()
	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().Delete(mock.Anything, oldTeamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *domain.TeamAuditLog) bool {
		return log.Action == domain.TeamActionDeleted
	})).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, &newTeamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *domain.TeamAuditLog) bool {
		return log.Action == domain.TeamActionJoined
	})).Return(nil).Once()

	uc := d.createUseCase()

	result, err := uc.Join(context.Background(), inviteToken, userID, true)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newTeamID, result.ID)
}

func TestTeamUseCase_Leave_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	captainID := uuid.New()
	teamID := uuid.New()

	user := &domain.User{
		ID:     userID,
		TeamID: &teamID,
	}

	team := &domain.Team{
		ID:        teamID,
		CaptainID: captainID,
	}

	members := []*domain.User{user, {ID: captainID, TeamID: &teamID}}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *domain.TeamAuditLog) bool {
		return log.Action == domain.TeamActionLeft
	})).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.Leave(context.Background(), userID)

	assert.NoError(t, err)
}

func TestTeamUseCase_Leave_CaptainCannotLeave_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()

	captain := &domain.User{
		ID:     captainID,
		TeamID: &teamID,
	}

	team := &domain.Team{
		ID:        teamID,
		CaptainID: captainID,
	}

	members := []*domain.User{captain, {ID: uuid.New(), TeamID: &teamID}}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Once()

	uc := d.createUseCase()

	err := uc.Leave(context.Background(), captainID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrCaptainCannotLeave)
}

func TestTeamUseCase_Leave_TeamBelowMinSize_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	captainID := uuid.New()
	teamID := uuid.New()

	user := &domain.User{ID: userID, TeamID: &teamID}
	team := &domain.Team{ID: teamID, CaptainID: captainID}
	members := []*domain.User{user, {ID: captainID, TeamID: &teamID}} // 2 members

	comp := &domain.Competition{Mode: "teams_only", AllowTeamSwitch: true, MinTeamSize: 2}
	d.compRepo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(comp, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Once()

	uc := d.createUseCase()

	err := uc.Leave(context.Background(), userID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamBelowMinSize)
}
