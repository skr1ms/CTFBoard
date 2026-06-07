package team

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func TestTeamUseCase_TryCreate_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	user := newTestUser(captainID, nil, "captain", "captain@example.com")

	comp := &domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}
	d.compRepo.EXPECT().Get(mock.Anything).Return(comp, nil).Times(2)
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{}, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "TestTeam").Return(nil, apperr.ErrTeamNotFound).Once()
	d.teamRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t *domain.Team) bool {
		return t.Name == "TestTeam" && t.CaptainID == captainID
	})).Return(nil).Run(func(_ context.Context, team *domain.Team) {
		team.ID = uuid.New()
		team.InviteToken = uuid.New()
	}).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, captainID, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.Anything).Return(nil).Once()

	uc := d.createUseCase()

	result, err := uc.TryCreate(context.Background(), "TestTeam", captainID, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Team)
	assert.Equal(t, "TestTeam", result.Team.Name)
}

func TestTeamUseCase_TryCreate_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	d.compRepo.EXPECT().Get(mock.Anything).Return(nil, assert.AnError).Once()

	uc := d.createUseCase()

	result, err := uc.TryCreate(context.Background(), "Team", uuid.New(), false)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTeamUseCase_ConfirmCreate_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	user := newTestUser(captainID, nil, "captain", "captain@example.com")

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "ConfirmTeam").Return(nil, apperr.ErrTeamNotFound).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t *domain.Team) bool {
		return t.Name == "ConfirmTeam" && t.CaptainID == captainID
	})).Return(nil).Run(func(_ context.Context, team *domain.Team) {
		team.ID = uuid.New()
		team.InviteToken = uuid.New()
	}).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, captainID, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.Anything).Return(nil).Once()

	uc := d.createUseCase()

	team, err := uc.ConfirmCreate(context.Background(), "ConfirmTeam", captainID, false)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, "ConfirmTeam", team.Name)
}

func TestTeamUseCase_ConfirmCreate_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(nil, assert.AnError).Once()

	uc := d.createUseCase()

	team, err := uc.ConfirmCreate(context.Background(), "Team", uuid.New(), false)

	assert.Error(t, err)
	assert.Nil(t, team)
}

func TestTeamUseCase_TryCreate_RequiresConfirmation(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	oldTeamID := uuid.New()
	user := newTestUser(captainID, &oldTeamID, "cap", "cap@x.com")
	oldTeam := &domain.Team{ID: oldTeamID, IsSolo: true, CaptainID: captainID}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Times(2)
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, oldTeamID).Return(oldTeam, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, oldTeamID).Return([]*domain.User{user}, nil).Once()
	d.solveRepo.EXPECT().GetTeamScore(mock.Anything, oldTeamID).Return(100, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, oldTeamID).Return([]*domain.SolveWithDetails{}, nil).Maybe()
	d.awardRepo.EXPECT().GetTeamTotalAwards(mock.Anything, oldTeamID).Return(0, nil).Maybe()

	uc := d.createUseCase()

	result, err := uc.TryCreate(context.Background(), "NewTeam", captainID, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.RequiresConfirm)
	assert.Equal(t, usecase.ConfirmReasonSoloTeamReset, result.ConfirmationReason)
	require.NotNil(t, result.AffectedData)
	assert.Equal(t, 100, result.AffectedData.Points)
}
