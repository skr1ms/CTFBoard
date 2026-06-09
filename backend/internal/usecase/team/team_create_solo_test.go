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

type soloScoreboardInvalidator struct {
	forTeamCalls int
	teamID       uuid.UUID
}

func (s *soloScoreboardInvalidator) InvalidateAll(context.Context) {}

func (s *soloScoreboardInvalidator) InvalidateForTeam(_ context.Context, teamID uuid.UUID) {
	s.forTeamCalls++
	s.teamID = teamID
}

func (s *soloScoreboardInvalidator) InvalidateLiveOnly(context.Context, uuid.UUID) {}

func TestTeamUseCase_CreateSoloTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)
	scoreboardCache := &soloScoreboardInvalidator{}
	d.scoreboardCache = scoreboardCache

	userID := uuid.New()
	user := &domain.User{ID: userID, Username: "solo_user"}
	createdTeamID := uuid.New()

	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "solo_only", AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "solo_user").Return(nil, apperr.ErrTeamNotFound).Once()

	d.teamRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t *domain.Team) bool {
		return t.IsSolo == true && t.CaptainID == userID && t.Name == "solo_user"
	})).Return(nil).Run(func(_ context.Context, team *domain.Team) {
		team.ID = createdTeamID
	}).Once()
	d.teamRepo.EXPECT().UpdateInviteToken(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *domain.TeamAuditLog) bool {
		return log.Action == domain.TeamActionCreated
	})).Return(nil).Once()

	uc := d.createUseCase()
	team, err := uc.CreateSoloTeam(context.Background(), userID, false)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.True(t, team.IsSolo)
	assert.Equal(t, "solo_user", team.Name)
	assert.Equal(t, team.ID, team.InviteToken)
	assert.Nil(t, team.InviteTokenExpiresAt)
	assert.Equal(t, 1, scoreboardCache.forTeamCalls)
	assert.Equal(t, createdTeamID, scoreboardCache.teamID)
}

func TestTeamUseCase_CreateSoloTeamForNewUser_InvalidatesScoreboard(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)
	scoreboardCache := &soloScoreboardInvalidator{}
	d.scoreboardCache = scoreboardCache

	userID := uuid.New()
	user := &domain.User{ID: userID, Username: "solo_new"}
	createdTeamID := uuid.New()

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: domain.ModeSoloOnly}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: domain.ModeSoloOnly, AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "solo_new").Return(nil, apperr.ErrTeamNotFound).Once()
	d.teamRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t *domain.Team) bool {
		return t.IsSolo && t.IsAutoCreated && t.CaptainID == userID && t.Name == "solo_new"
	})).Return(nil).Run(func(_ context.Context, team *domain.Team) {
		team.ID = createdTeamID
	}).Once()
	d.teamRepo.EXPECT().UpdateInviteToken(mock.Anything, createdTeamID, createdTeamID, mock.Anything).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *domain.TeamAuditLog) bool {
		return log.Action == domain.TeamActionCreated
	})).Return(nil).Once()

	uc := d.createUseCase()
	team, err := uc.CreateSoloTeamForNewUser(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, 1, scoreboardCache.forTeamCalls)
	assert.Equal(t, createdTeamID, scoreboardCache.teamID)
}

func TestTeamUseCase_CreateSoloTeam_WasInBannedTeam_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	user := &domain.User{ID: userID, Username: "banned_user", WasInBannedTeam: true}

	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "solo_only", AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()

	uc := d.createUseCase()
	team, err := uc.CreateSoloTeam(context.Background(), userID, false)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrUserWasInBannedTeam)
	assert.Nil(t, team)
}

func TestTeamUseCase_CreateSoloTeam_Error_AlreadyInTeam(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	teamID := uuid.New()
	user := &domain.User{ID: userID, TeamID: &teamID}

	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "solo_only", AllowTeamSwitch: true}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsSolo: false, IsAutoCreated: false}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{user}, nil).Once()

	uc := d.createUseCase()
	team, err := uc.CreateSoloTeam(context.Background(), userID, false)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrUserAlreadyInTeam)
	assert.Nil(t, team)
}
