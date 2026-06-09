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

func TestTeamUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	user := newTestUser(captainID, nil, "captain", "captain@example.com")

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "TestTeam").Return(nil, apperr.ErrTeamNotFound).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t *domain.Team) bool {
		return t.Name == "TestTeam" && t.CaptainID == captainID
	})).Return(nil).Run(func(_ context.Context, team *domain.Team) {
		team.ID = uuid.New()
		team.InviteToken = uuid.New()
	}).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, captainID, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.Anything).Return(nil).Once()

	uc := d.createUseCase()

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, "TestTeam", team.Name)
	assert.Equal(t, captainID, team.CaptainID)
	assert.NotEmpty(t, team.InviteToken)
}

func TestTeamUseCase_Create_WithSoloTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	oldTeamID := uuid.New()
	user := newTestUser(captainID, &oldTeamID, "captain", "captain@example.com")

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "NewTeam").Return(nil, apperr.ErrTeamNotFound).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, oldTeamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, oldTeamID).Return(&domain.Team{ID: oldTeamID, IsSolo: true}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, oldTeamID).Return([]*domain.User{user}, nil).Once()
	d.solveRepo.EXPECT().GetModerationAffectedChallengeIDsByTeamID(mock.Anything, oldTeamID).Return([]uuid.UUID{}, nil).Once()
	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, captainID, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().Delete(mock.Anything, oldTeamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *domain.TeamAuditLog) bool {
		return log.Action == domain.TeamActionDeleted
	})).Return(nil).Once()
	d.teamRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, team *domain.Team) {
		team.ID = uuid.New()
		team.InviteToken = uuid.New()
	}).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, captainID, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *domain.TeamAuditLog) bool {
		return log.Action == domain.TeamActionCreated
	})).Return(nil).Once()

	uc := d.createUseCase()

	team, err := uc.Create(context.Background(), "NewTeam", captainID, false, true)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, "NewTeam", team.Name)
}

func TestTeamUseCase_Create_TeamNameExists_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	existingTeam := &domain.Team{
		ID:   uuid.New(),
		Name: "TestTeam",
	}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "TestTeam").Return(existingTeam, nil).Once()

	uc := d.createUseCase()

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.Error(t, err)
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_UserAlreadyInMultiMemberTeam_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	otherUserID := uuid.New()
	user := &domain.User{
		ID:     captainID,
		TeamID: &teamID,
	}
	otherUser := &domain.User{
		ID:     otherUserID,
		TeamID: &teamID,
	}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "TestTeam").Return(nil, apperr.ErrTeamNotFound).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsSolo: false}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{user, otherUser}, nil).Once()

	uc := d.createUseCase()

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrUserAlreadyInTeam)
	assert.Nil(t, team)
}

func TestTeamUseCase_Create_MaxTeamsReached_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{MaxTeams: 1}, nil).Once()
	d.teamRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().CountActiveTeams(mock.Anything).Return(1, nil).Once()

	uc := d.createUseCase()

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrMaxTeamsReached)
	assert.Nil(t, team)
}
