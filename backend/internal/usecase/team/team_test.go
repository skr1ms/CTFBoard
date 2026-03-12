package team

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type teamTestDeps struct {
	teamRepo       *mocks.MockTeamRepository
	userRepo       *mocks.MockUserRepository
	solveRepo      *mocks.MockSolveRepository
	submissionRepo *mocks.MockSubmissionRepository
	awardRepo      *mocks.MockAwardRepository
	compRepo       *mocks.MockCompetitionRepository
	SettingsRepo   *mocks.MockSettingsRepository
	challengeRepo  *mocks.MockChallengeRepository
	tm             *mocks.MockTransactionManager
}

func newTeamTestDeps(t *testing.T) *teamTestDeps {
	t.Helper()
	return &teamTestDeps{
		teamRepo:       mocks.NewMockTeamRepository(t),
		userRepo:       mocks.NewMockUserRepository(t),
		solveRepo:      mocks.NewMockSolveRepository(t),
		submissionRepo: mocks.NewMockSubmissionRepository(t),
		awardRepo:      mocks.NewMockAwardRepository(t),
		compRepo:       mocks.NewMockCompetitionRepository(t),
		SettingsRepo:   mocks.NewMockSettingsRepository(t),
		challengeRepo:  mocks.NewMockChallengeRepository(t),
		tm:             mocks.NewMockTransactionManager(t),
	}
}

func (d *teamTestDeps) createUseCase() *TeamUseCase {
	return NewTeamUseCase(TeamDeps{
		TeamRepo: d.teamRepo, UserRepo: d.userRepo, SolveRepo: d.solveRepo,
		SubmissionRepo: d.submissionRepo, AwardRepo: d.awardRepo, CompRepo: d.compRepo,
		SettingsGetter: d.SettingsRepo, ChallengeRepo: d.challengeRepo, TM: d.tm,
		Guard: competition.NewGuard(d.compRepo), DefaultMaxTeamSize: 10,
	})
}

func newTestUser(id uuid.UUID, teamID *uuid.UUID, username, email string) *entity.User {
	return &entity.User{ID: id, Username: username, Email: email, TeamID: teamID}
}

func newTestTeam(id uuid.UUID, name string, captainID, inviteToken uuid.UUID, isSolo bool) *entity.Team {
	return &entity.Team{ID: id, Name: name, CaptainID: captainID, InviteToken: inviteToken, IsSolo: isSolo}
}

func TestTeamUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	user := newTestUser(captainID, nil, "captain", "captain@example.com")

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "TestTeam").Return(nil, httperr.ErrTeamNotFound).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t *entity.Team) bool {
		return t.Name == "TestTeam" && t.CaptainID == captainID
	})).Return(nil).Run(func(_ context.Context, team *entity.Team) {
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
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "NewTeam").Return(nil, httperr.ErrTeamNotFound).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, oldTeamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, oldTeamID).Return(&entity.Team{ID: oldTeamID, IsSolo: true}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, oldTeamID).Return([]*entity.User{user}, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, oldTeamID).Return([]*entity.SolveWithDetails{}, nil).Once()
	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, captainID, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().Delete(mock.Anything, oldTeamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionDeleted
	})).Return(nil).Once()
	d.teamRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, team *entity.Team) {
		team.ID = uuid.New()
		team.InviteToken = uuid.New()
	}).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, captainID, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionCreated
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
	existingTeam := &entity.Team{
		ID:   uuid.New(),
		Name: "TestTeam",
	}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{}, nil).Once()
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
	user := &entity.User{
		ID:     captainID,
		TeamID: &teamID,
	}
	otherUser := &entity.User{
		ID:     otherUserID,
		TeamID: &teamID,
	}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "TestTeam").Return(nil, httperr.ErrTeamNotFound).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsSolo: false}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*entity.User{user, otherUser}, nil).Once()

	uc := d.createUseCase()

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrUserAlreadyInTeam))
	assert.Nil(t, team)
}

func TestTeamUseCase_Join_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

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

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Times(3)
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByInviteToken(mock.Anything, inviteToken).Return(team, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*entity.User{}, nil).Times(2)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, &teamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionJoined
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

	team := &entity.Team{
		ID:          teamID,
		Name:        "TestTeam",
		InviteToken: inviteToken,
	}

	existingMembers := make([]*entity.User, 10)
	for i := 0; i < 10; i++ {
		existingMembers[i] = &entity.User{ID: uuid.New()}
	}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true, MaxTeamSize: 10}, nil).Times(2)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByInviteToken(mock.Anything, inviteToken).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(existingMembers, nil).Once()

	uc := d.createUseCase()

	result, err := uc.Join(context.Background(), inviteToken, userID, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamFull))
	assert.Nil(t, result)
}

func TestTeamUseCase_Join_WithSoloTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

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

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Times(3)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByInviteToken(mock.Anything, inviteToken).Return(newTeam, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, mock.Anything).Return(nil).Times(4)
	d.teamRepo.EXPECT().GetByID(mock.Anything, newTeamID).Return(newTeam, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, newTeamID).Return([]*entity.User{}, nil).Times(2)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, oldTeamID).Return(&entity.Team{ID: oldTeamID, IsSolo: true}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, oldTeamID).Return([]*entity.User{user}, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, oldTeamID).Return([]*entity.SolveWithDetails{}, nil).Once()
	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, oldTeamID).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().Delete(mock.Anything, oldTeamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionDeleted
	})).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, &newTeamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionJoined
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

	user := &entity.User{
		ID:     userID,
		TeamID: &teamID,
	}

	team := &entity.Team{
		ID:        teamID,
		CaptainID: captainID,
	}

	members := []*entity.User{user, {ID: captainID, TeamID: &teamID}}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Times(2)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionLeft
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

	captain := &entity.User{
		ID:     captainID,
		TeamID: &teamID,
	}

	team := &entity.Team{
		ID:        teamID,
		CaptainID: captainID,
	}

	members := []*entity.User{captain, {ID: uuid.New(), TeamID: &teamID}}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Times(2)
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
	assert.True(t, errors.Is(err, httperr.ErrCaptainCannotLeave))
}

func TestTeamUseCase_Leave_TeamBelowMinSize_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	captainID := uuid.New()
	teamID := uuid.New()

	user := &entity.User{ID: userID, TeamID: &teamID}
	team := &entity.Team{ID: teamID, CaptainID: captainID}
	members := []*entity.User{user, {ID: captainID, TeamID: &teamID}} // 2 members

	comp := &entity.Competition{Mode: "flexible", AllowTeamSwitch: true, MinTeamSize: 2}
	d.compRepo.EXPECT().Get(mock.Anything).Return(comp, nil).Times(2)
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
	assert.True(t, errors.Is(err, httperr.ErrTeamBelowMinSize))
}

func TestTeamUseCase_TransferCaptain_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

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

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Times(2)
	d.userRepo.EXPECT().Lock(mock.Anything, mock.Anything).Return(nil).Times(2)
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, newCaptainID).Return(newCaptain, nil).Once()
	d.teamRepo.EXPECT().UpdateCaptain(mock.Anything, teamID, newCaptainID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionCaptainTransfer
	})).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.TransferCaptain(context.Background(), captainID, newCaptainID)

	assert.NoError(t, err)
}

func TestTeamUseCase_TransferCaptain_NotCaptain_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

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

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Times(2)
	d.userRepo.EXPECT().Lock(mock.Anything, mock.Anything).Return(nil).Times(2)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	newCaptain := &entity.User{ID: newCaptainID, TeamID: &teamID}
	d.userRepo.EXPECT().GetByID(mock.Anything, newCaptainID).Return(newCaptain, nil).Once()

	uc := d.createUseCase()

	err := uc.TransferCaptain(context.Background(), userID, newCaptainID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrNotCaptain))
}

func TestTeamUseCase_GetByID_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	expectedTeam := &entity.Team{
		ID:          teamID,
		Name:        "TestTeam",
		InviteToken: uuid.New(),
		CaptainID:   uuid.New(),
	}

	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(expectedTeam, nil).Once()

	uc := d.createUseCase()

	team, err := uc.GetByID(context.Background(), teamID)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, expectedTeam.ID, team.ID)
	assert.Equal(t, expectedTeam.Name, team.Name)
}

func TestTeamUseCase_GetMyTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

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

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{MinTeamSize: 0}, nil).Once()

	uc := d.createUseCase()

	result, gotMembers, minSize, meetsMin, err := uc.GetMyTeam(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, teamID, result.ID)
	assert.Equal(t, "MyTeam", result.Name)
	assert.NotNil(t, gotMembers)
	assert.Equal(t, 1, len(gotMembers))
	assert.Equal(t, 0, minSize)
	assert.True(t, meetsMin)
}

func TestTeamUseCase_GetTeamMembers_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

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

	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamMembers(context.Background(), teamID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
}

func TestTeamUseCase_CreateSoloTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	user := &entity.User{ID: userID, Username: "solo_user"}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "solo_only", AllowTeamSwitch: true}, nil).Times(3)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "solo_user").Return(nil, httperr.ErrTeamNotFound).Once()

	d.teamRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t *entity.Team) bool {
		return t.IsSolo == true && t.CaptainID == userID && t.Name == "solo_user"
	})).Return(nil).Run(func(_ context.Context, team *entity.Team) {
		team.ID = uuid.New()
	}).Once()
	d.teamRepo.EXPECT().UpdateInviteToken(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(log *entity.TeamAuditLog) bool {
		return log.Action == entity.TeamActionCreated
	})).Return(nil).Once()

	uc := d.createUseCase()
	team, err := uc.CreateSoloTeam(context.Background(), userID, false)

	assert.NoError(t, err)
	assert.NotNil(t, team)
	assert.True(t, team.IsSolo)
	assert.Equal(t, "solo_user", team.Name)
	assert.Equal(t, team.ID, team.InviteToken)
	assert.Nil(t, team.InviteTokenExpiresAt)
}

func TestTeamUseCase_CreateSoloTeam_WasInBannedTeam_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	user := &entity.User{ID: userID, Username: "banned_user", WasInBannedTeam: true}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "solo_only", AllowTeamSwitch: true}, nil).Times(3)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()

	uc := d.createUseCase()
	team, err := uc.CreateSoloTeam(context.Background(), userID, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrUserWasInBannedTeam))
	assert.Nil(t, team)
}

func TestTeamUseCase_CreateSoloTeam_Error_AlreadyInTeam(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	teamID := uuid.New()
	user := &entity.User{ID: userID, TeamID: &teamID}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "solo_only", AllowTeamSwitch: true}, nil).Times(3)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{}, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsSolo: false, IsAutoCreated: false}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*entity.User{user}, nil).Once()

	uc := d.createUseCase()
	team, err := uc.CreateSoloTeam(context.Background(), userID, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrUserAlreadyInTeam))
	assert.Nil(t, team)
}

func TestTeamUseCase_RosterFrozen_BlocksAllOperations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		setup  func(deps *teamTestDeps)
		action func(uc *TeamUseCase) error
	}{
		{"Create", func(deps *teamTestDeps) {
			deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()
			deps.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()
		}, func(uc *TeamUseCase) error {
			_, err := uc.Create(context.Background(), "test_team", uuid.New(), false, false)
			return err
		}},
		{"Join", nil, func(uc *TeamUseCase) error {
			_, err := uc.Join(context.Background(), uuid.New(), uuid.New(), false)
			return err
		}},
		{"CreateSoloTeam", nil, func(uc *TeamUseCase) error {
			_, err := uc.CreateSoloTeam(context.Background(), uuid.New(), false)
			return err
		}},
		{"Leave", nil, func(uc *TeamUseCase) error { return uc.Leave(context.Background(), uuid.New()) }},
		{"TransferCaptain", nil, func(uc *TeamUseCase) error {
			return uc.TransferCaptain(context.Background(), uuid.New(), uuid.New())
		}},
		{"DisbandTeam", nil, func(uc *TeamUseCase) error { return uc.DisbandTeam(context.Background(), uuid.New()) }},
		{"KickMember", nil, func(uc *TeamUseCase) error { return uc.KickMember(context.Background(), uuid.New(), uuid.New()) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := newTeamTestDeps(t)
			if tc.setup != nil {
				tc.setup(d)
			} else {
				d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{AllowTeamSwitch: false}, nil).Once()
			}
			uc := d.createUseCase()
			err := tc.action(uc)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, httperr.ErrRosterFrozen))
		})
	}
}

func TestTeamUseCase_DisbandTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	captain := &entity.User{ID: captainID, TeamID: &teamID}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: entity.ModeFlexible, AllowTeamSwitch: true}, nil).Times(2)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, CaptainID: captainID, Name: "test_team"}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*entity.User{captain}, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return(nil, nil).Once()
	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		return l.Action == entity.TeamActionDeleted && l.TeamID == teamID && l.UserID != nil && *l.UserID == captainID
	})).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamIDBatch(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 1 && ids[0] == captainID
	}), mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().Delete(mock.Anything, teamID).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.DisbandTeam(context.Background(), captainID)

	assert.NoError(t, err)
}

func TestTeamUseCase_DisbandTeam_WithSolves_UsesGetByIDs(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	ch1ID := uuid.New()
	ch2ID := uuid.New()
	captain := &entity.User{ID: captainID, TeamID: &teamID}

	solvesWithDetails := []*entity.SolveWithDetails{
		{Solve: entity.Solve{ChallengeID: ch1ID}, ChallengePoints: 100, ChallengeTitle: "Ch1"},
		{Solve: entity.Solve{ChallengeID: ch1ID}, ChallengePoints: 100, ChallengeTitle: "Ch1"},
		{Solve: entity.Solve{ChallengeID: ch2ID}, ChallengePoints: 200, ChallengeTitle: "Ch2"},
	}
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: entity.ModeFlexible, AllowTeamSwitch: true}, nil).Times(2)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, CaptainID: captainID, Name: "test_team"}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*entity.User{captain}, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return(solvesWithDetails, nil).Once()
	d.challengeRepo.EXPECT().BatchDecrementSolveCount(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 2 && (ids[0] == ch1ID && ids[1] == ch2ID || ids[0] == ch2ID && ids[1] == ch1ID)
	})).Return(nil).Once()
	ch1After := &entity.Challenge{ID: ch1ID, InitialValue: 100, MinValue: 50, Decay: 10, SolveCount: 0}
	ch2After := &entity.Challenge{ID: ch2ID, InitialValue: 200, MinValue: 100, Decay: 20, SolveCount: 0}
	d.challengeRepo.EXPECT().GetByIDs(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 2 && (ids[0] == ch1ID && ids[1] == ch2ID || ids[0] == ch2ID && ids[1] == ch1ID)
	})).Return(map[uuid.UUID]*entity.Challenge{ch1ID: ch1After, ch2ID: ch2After}, nil).Once()
	d.challengeRepo.EXPECT().BatchUpdatePoints(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 2
	}), mock.MatchedBy(func(points []int) bool {
		return len(points) == 2 && ((points[0] == 100 && points[1] == 200) || (points[0] == 200 && points[1] == 100))
	})).Return(nil).Once()
	d.solveRepo.EXPECT().GetSolvesForPointsRecalc(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 2 && (ids[0] == ch1ID && ids[1] == ch2ID || ids[0] == ch2ID && ids[1] == ch1ID)
	})).Return(nil, nil).Once()
	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		return l.Action == entity.TeamActionDeleted && l.TeamID == teamID && l.UserID != nil && *l.UserID == captainID
	})).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamIDBatch(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 1 && ids[0] == captainID
	}), mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().Delete(mock.Anything, teamID).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.DisbandTeam(context.Background(), captainID)

	assert.NoError(t, err)
}

func TestTeamUseCase_DisbandTeam_BannedCaptain_ReturnsErrUserBanned(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	captain := &entity.User{ID: captainID, TeamID: &teamID, IsBanned: true}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: entity.ModeFlexible, AllowTeamSwitch: true}, nil).Times(2)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, CaptainID: captainID, Name: "test_team"}, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*entity.User{captain}, nil).Once()

	uc := d.createUseCase()
	err := uc.DisbandTeam(context.Background(), captainID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrUserBanned))
}

func TestTeamUseCase_KickMember_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	targetID := uuid.New()
	teamID := uuid.New()
	captain := &entity.User{ID: captainID, TeamID: &teamID}
	target := &entity.User{ID: targetID, TeamID: &teamID}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Times(2)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, mock.Anything).Return(nil).Twice()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, CaptainID: captainID, Name: "test_team"}, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, targetID).Return(target, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*entity.User{captain, target}, nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, targetID, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		targetIDStr := targetID.String()
		detailsTargetID, ok := l.Details["target_user_id"].(string)
		return l.Action == entity.TeamActionMemberKicked &&
			l.TeamID == teamID &&
			l.UserID != nil && *l.UserID == captainID &&
			ok && detailsTargetID == targetIDStr
	})).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.KickMember(context.Background(), captainID, targetID)

	assert.NoError(t, err)
}

func TestTeamUseCase_GetByID_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, httperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	team, err := uc.GetByID(context.Background(), teamID)

	assert.Error(t, err)
	assert.Nil(t, team)
	assert.True(t, errors.Is(err, httperr.ErrTeamNotFound))
}

func TestTeamUseCase_GetMyTeam_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, httperr.ErrUserNotFound).Once()

	uc := d.createUseCase()

	team, members, _, _, err := uc.GetMyTeam(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, team)
	assert.Nil(t, members)
	assert.True(t, errors.Is(err, httperr.ErrUserNotFound))
}

func TestTeamUseCase_GetTeamMembers_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(nil, errors.New("db error")).Once()

	uc := d.createUseCase()

	members, err := uc.GetTeamMembers(context.Background(), teamID)

	assert.Error(t, err)
	assert.Nil(t, members)
}

func TestTeamUseCase_BanTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	team := &entity.Team{ID: teamID, Name: "Team"}
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*entity.User{}, nil).Twice()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().Ban(mock.Anything, teamID, "reason").Return(nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return([]*entity.SolveWithDetails{}, nil).Once()
	d.solveRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamIDBatch(mock.Anything, []uuid.UUID{}, (*uuid.UUID)(nil)).Return(nil).Once()
	actorID := uuid.New()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		return l.TeamID == teamID && l.UserID != nil && *l.UserID == actorID && l.Action == entity.TeamActionBanned && l.Details["reason"] == "reason"
	})).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.BanTeam(context.Background(), teamID, "reason", false, actorID)

	assert.NoError(t, err)
}

func TestTeamUseCase_BanTeam_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	userID := uuid.New()
	members := []*entity.User{{ID: userID}}
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Twice()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, httperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	err := uc.BanTeam(context.Background(), teamID, "reason", false, uuid.Nil)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamNotFound))
}

func TestTeamUseCase_UnbanTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	team := &entity.Team{ID: teamID, Name: "Team", IsBanned: true}
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().GetLatestAuditLogByTeamIDAndAction(mock.Anything, teamID, "banned").Return(nil, nil).Once()
	d.userRepo.EXPECT().FilterIDsByTeamIDNullAndNotBanned(mock.Anything, mock.Anything).Return([]uuid.UUID(nil), nil).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{MaxTeamSize: 10}, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().Unban(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, true).Return(nil).Once()
	actorID := uuid.New()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		return l.TeamID == teamID && l.UserID != nil && *l.UserID == actorID && l.Action == entity.TeamActionUnbanned
	})).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.UnbanTeam(context.Background(), teamID, actorID)

	assert.NoError(t, err)
}

func TestTeamUseCase_UnbanTeam_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().GetLatestAuditLogByTeamIDAndAction(mock.Anything, teamID, "banned").Return(nil, errors.New("db error")).Once()

	uc := d.createUseCase()

	err := uc.UnbanTeam(context.Background(), teamID, uuid.Nil)

	assert.Error(t, err)
}

func TestTeamUseCase_UnbanTeam_DoesNotUnbanIndependentlyBannedMember(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	userA := uuid.New()
	userB := uuid.New()
	team := &entity.Team{ID: teamID, Name: "Team", CaptainID: userA, IsBanned: true}
	banLog := &entity.TeamAuditLog{
		TeamID:  teamID,
		Action:  entity.TeamActionBanned,
		Details: map[string]any{"reason": "cheat", "member_ids": []string{userA.String(), userB.String()}, "ban_members": true, "banned_user_ids": []string{userA.String()}},
	}
	userAModel := &entity.User{ID: userA, IsBanned: true}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().GetLatestAuditLogByTeamIDAndAction(mock.Anything, teamID, "banned").Return(banLog, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userA).Return(nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userB).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userA).Return(userAModel, nil).Once()
	d.userRepo.EXPECT().Unban(mock.Anything, userA).Return(nil).Once()
	memberIDsMatcher := mock.MatchedBy(func(ids []uuid.UUID) bool {
		return len(ids) == 2 && ((ids[0] == userA && ids[1] == userB) || (ids[0] == userB && ids[1] == userA))
	})
	d.userRepo.EXPECT().FilterIDsByTeamIDNullAndNotBanned(mock.Anything, memberIDsMatcher).Return([]uuid.UUID{userA}, nil).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{MaxTeamSize: 10}, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().Unban(mock.Anything, teamID).Return(nil).Once()
	d.solveRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return([]*entity.SolveWithDetails{}, nil).Once()
	d.userRepo.EXPECT().FilterIDsByTeamIDNullAndNotBanned(mock.Anything, memberIDsMatcher).Return([]uuid.UUID{userA}, nil).Once()
	d.userRepo.EXPECT().UpdateTeamIDBatch(mock.Anything, []uuid.UUID{userA}, &teamID).Return(nil).Once()
	d.userRepo.EXPECT().SetWasInBannedTeamByIDs(mock.Anything, memberIDsMatcher, false).Return(nil).Once()
	actorID := uuid.New()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		return l.TeamID == teamID && l.Action == entity.TeamActionUnbanned
	})).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.UnbanTeam(context.Background(), teamID, actorID)

	assert.NoError(t, err)
}

func TestTeamUseCase_SetHidden_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	team := &entity.Team{ID: teamID, Name: "Team"}
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, true).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.SetHidden(context.Background(), teamID, true)

	assert.NoError(t, err)
}

func TestTeamUseCase_SetHidden_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, httperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	err := uc.SetHidden(context.Background(), teamID, true)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamNotFound))
}

func TestTeamUseCase_SetBracket_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	bracketID := uuid.New()
	team := &entity.Team{ID: teamID, Name: "Team"}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().SetBracket(mock.Anything, teamID, &bracketID).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.SetBracket(context.Background(), teamID, &bracketID)

	assert.NoError(t, err)
}

func TestTeamUseCase_SetBracket_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, httperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	err := uc.SetBracket(context.Background(), teamID, nil)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamNotFound))
}

func TestTeamUseCase_TryCreate_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	user := newTestUser(captainID, nil, "captain", "captain@example.com")

	comp := &entity.Competition{Mode: "flexible", AllowTeamSwitch: true}
	d.compRepo.EXPECT().Get(mock.Anything).Return(comp, nil).Times(2)
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{}, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "TestTeam").Return(nil, httperr.ErrTeamNotFound).Once()
	d.teamRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t *entity.Team) bool {
		return t.Name == "TestTeam" && t.CaptainID == captainID
	})).Return(nil).Run(func(_ context.Context, team *entity.Team) {
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

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "ConfirmTeam").Return(nil, httperr.ErrTeamNotFound).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t *entity.Team) bool {
		return t.Name == "ConfirmTeam" && t.CaptainID == captainID
	})).Return(nil).Run(func(_ context.Context, team *entity.Team) {
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

func TestTeamUseCase_ListTeams_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	d.tm.EXPECT().ReadOnly(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	teams := []*entity.Team{{ID: uuid.New(), Name: "Team1"}}
	d.teamRepo.EXPECT().Search(mock.Anything, (*string)(nil), 10, 0).Return(teams, nil).Once()
	d.teamRepo.EXPECT().CountSearch(mock.Anything, (*string)(nil)).Return(int64(1), nil).Once()

	uc := d.createUseCase()

	result, err := uc.ListTeams(context.Background(), nil, 1, 10)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Data, 1)
	assert.Equal(t, int64(1), result.Total)
}

func TestTeamUseCase_ListTeams_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	d.tm.EXPECT().ReadOnly(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Search(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("search failed")).Maybe()
	d.teamRepo.EXPECT().CountSearch(mock.Anything, mock.Anything).Return(int64(0), errors.New("count failed")).Maybe()

	uc := d.createUseCase()

	_, err := uc.ListTeams(context.Background(), nil, 1, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ListTeams")
}

func TestTeamUseCase_AdminListTeams_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	d.tm.EXPECT().ReadOnly(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	teams := []*entity.Team{{ID: uuid.New(), Name: "AdminTeam1"}}
	d.teamRepo.EXPECT().SearchAdmin(mock.Anything, (*string)(nil), 10, 0).Return(teams, nil).Once()
	d.teamRepo.EXPECT().CountSearchAdmin(mock.Anything, (*string)(nil)).Return(int64(1), nil).Once()

	uc := d.createUseCase()

	result, err := uc.AdminListTeams(context.Background(), nil, 1, 10)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Data, 1)
}

func TestTeamUseCase_AdminListTeams_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	d.tm.EXPECT().ReadOnly(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().SearchAdmin(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("admin search failed")).Maybe()
	d.teamRepo.EXPECT().CountSearchAdmin(mock.Anything, mock.Anything).Return(int64(0), errors.New("count failed")).Maybe()

	uc := d.createUseCase()

	_, err := uc.AdminListTeams(context.Background(), nil, 1, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AdminListTeams")
}

func TestTeamUseCase_AdminUpdate_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	newName := "UpdatedName"
	updatedTeam := &entity.Team{ID: teamID, Name: newName}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(updatedTeam, nil).Once()
	d.teamRepo.EXPECT().UpdateAdmin(mock.Anything, teamID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(updatedTeam, nil).Once()

	uc := d.createUseCase()

	team, err := uc.AdminUpdate(context.Background(), teamID, &newName, nil, nil, nil)

	require.NoError(t, err)
	assert.Equal(t, newName, team.Name)
}

func TestTeamUseCase_AdminUpdate_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	newName := "Updated"

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).Return(errors.New("tx failed")).Once()

	uc := d.createUseCase()

	_, err := uc.AdminUpdate(context.Background(), teamID, &newName, nil, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tx failed")
}

func TestTeamUseCase_AdminDelete_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	memberID := uuid.New()
	members := []*entity.User{{ID: memberID, TeamID: &teamID}}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, memberID).Return(nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return([]*entity.SolveWithDetails{}, nil).Once()
	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamIDBatch(mock.Anything, []uuid.UUID{memberID}, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().Delete(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		return l.TeamID == teamID && l.Action == entity.TeamActionDeleted && l.Details["reason"] == "deleted_by_admin"
	})).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.AdminDelete(context.Background(), teamID)

	require.NoError(t, err)
}

func TestTeamUseCase_AdminDelete_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).Return(errors.New("tx error")).Once()

	uc := d.createUseCase()

	err := uc.AdminDelete(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tx error")
}

func TestTeamUseCase_AdminAddMember_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	userID := uuid.New()
	user := newTestUser(userID, nil, "member", "m@x.com")

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	nonSoloTeam := &entity.Team{ID: teamID, IsSolo: false}
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nonSoloTeam, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*entity.User{}, nil).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", MaxTeamSize: 5}, nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, &teamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		return l.TeamID == teamID && l.UserID != nil && *l.UserID == userID && l.Action == entity.TeamActionJoined
	})).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.AdminAddMember(context.Background(), teamID, userID)

	require.NoError(t, err)
}

func TestTeamUseCase_AdminAddMember_Error_UserAlreadyInTeam(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	otherTeamID := uuid.New()
	userID := uuid.New()
	user := newTestUser(userID, &otherTeamID, "member", "m@x.com")

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	nonSoloTeam := &entity.Team{ID: teamID, IsSolo: false}
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nonSoloTeam, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()

	uc := d.createUseCase()

	err := uc.AdminAddMember(context.Background(), teamID, userID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrTeamConflict)
}

func TestTeamUseCase_AdminRemoveMember_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	captainID := uuid.New()
	memberID := uuid.New()
	member := newTestUser(memberID, &teamID, "member", "m@x.com")
	team := newTestTeam(teamID, "Team", captainID, uuid.New(), false)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, memberID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, memberID).Return(member, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, memberID, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *entity.TeamAuditLog) bool {
		return l.TeamID == teamID && l.UserID != nil && *l.UserID == memberID && l.Action == entity.TeamActionMemberKicked
	})).Return(nil).Once()

	uc := d.createUseCase()

	err := uc.AdminRemoveMember(context.Background(), teamID, memberID)

	require.NoError(t, err)
}

func TestTeamUseCase_AdminRemoveMember_Error_CaptainCannotLeave(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	captainID := uuid.New()
	captain := newTestUser(captainID, &teamID, "captain", "c@x.com")
	team := newTestTeam(teamID, "Team", captainID, uuid.New(), false)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(captain, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()

	uc := d.createUseCase()

	err := uc.AdminRemoveMember(context.Background(), teamID, captainID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrCaptainCannotLeave)
}

func TestTeamUseCase_GetTeamSolves_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	solves := []*entity.SolveWithDetails{
		{
			Solve:          entity.Solve{ChallengeID: uuid.New(), SolvedAt: time.Now()},
			ChallengeTitle: "Ch1",
		},
	}

	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID}, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return(solves, nil).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamSolves(context.Background(), teamID)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, solves[0].ChallengeTitle, result[0].ChallengeTitle)
}

func TestTeamUseCase_GetTeamSolves_Error_TeamNotFound(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, httperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamSolves(context.Background(), teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamNotFound))
	assert.Nil(t, result)
}

func TestTeamUseCase_GetTeamFails_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	var fails []*entity.SubmissionWithDetails

	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID}, nil).Once()
	d.submissionRepo.EXPECT().GetFailsByTeam(mock.Anything, teamID, 10, 0).Return(fails, nil).Once()
	d.submissionRepo.EXPECT().CountFailsByTeam(mock.Anything, teamID).Return(int64(0), nil).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamFails(context.Background(), teamID, 1, 10)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Data, 0)
	assert.Equal(t, int64(0), result.Total)
}

func TestTeamUseCase_GetTeamFails_Error_TeamNotFound(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, httperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamFails(context.Background(), teamID, 1, 10)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamNotFound))
	assert.Nil(t, result)
}

func TestTeamUseCase_GetTeamAwards_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	awards := []*entity.Award{{ID: uuid.New(), TeamID: teamID, Value: 50}}

	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID}, nil).Once()
	d.awardRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(awards, nil).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamAwards(context.Background(), teamID)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 50, result[0].Value)
}

func TestTeamUseCase_GetTeamAwards_Error_TeamNotFound(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, httperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamAwards(context.Background(), teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamNotFound))
	assert.Nil(t, result)
}

func TestTeamUseCase_GetInviteToken_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	user := newTestUser(captainID, &teamID, "cap", "cap@x.com")
	team := newTestTeam(teamID, "MyTeam", captainID, uuid.New(), false)

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()

	uc := d.createUseCase()

	result, err := uc.GetInviteToken(context.Background(), captainID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, teamID, result.ID)
	assert.Equal(t, captainID, result.CaptainID)
}

func TestTeamUseCase_GetInviteToken_Error_NotCaptain(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	otherCaptainID := uuid.New()
	teamID := uuid.New()
	user := newTestUser(captainID, &teamID, "member", "m@x.com")
	team := newTestTeam(teamID, "MyTeam", otherCaptainID, uuid.New(), false)

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()

	uc := d.createUseCase()

	result, err := uc.GetInviteToken(context.Background(), captainID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrNotCaptain))
	assert.Nil(t, result)
}

func TestTeamUseCase_UpdateMyTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	user := newTestUser(captainID, &teamID, "cap", "cap@x.com")
	team := newTestTeam(teamID, "OldName", captainID, uuid.New(), false)

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Times(2)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "NewName").Return(nil, httperr.ErrTeamNotFound).Once()
	d.teamRepo.EXPECT().UpdateName(mock.Anything, teamID, "NewName").Return(nil).Once()

	uc := d.createUseCase()

	result, err := uc.UpdateMyTeam(context.Background(), captainID, "NewName")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "NewName", result.Name)
}

func TestTeamUseCase_UpdateMyTeam_Error_NotCaptain(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	realCaptainID := uuid.New()
	teamID := uuid.New()
	user := newTestUser(captainID, &teamID, "member", "m@x.com")
	team := newTestTeam(teamID, "Team", realCaptainID, uuid.New(), false)

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Times(2)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()

	uc := d.createUseCase()

	result, err := uc.UpdateMyTeam(context.Background(), captainID, "NewName")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrNotCaptain))
	assert.Nil(t, result)
}

func TestTeamUseCase_TryCreate_RequiresConfirmation(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	oldTeamID := uuid.New()
	user := newTestUser(captainID, &oldTeamID, "cap", "cap@x.com")
	oldTeam := &entity.Team{ID: oldTeamID, IsSolo: true, CaptainID: captainID}

	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Times(2)
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, oldTeamID).Return(oldTeam, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, oldTeamID).Return([]*entity.User{user}, nil).Once()
	d.solveRepo.EXPECT().GetTeamScore(mock.Anything, oldTeamID).Return(100, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, oldTeamID).Return([]*entity.SolveWithDetails{}, nil).Maybe()
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

func TestTeamUseCase_Create_MaxTeamsReached_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&entity.Competition{Mode: "flexible", AllowTeamSwitch: true}, nil).Once()
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{MaxTeams: 1}, nil).Once()
	d.teamRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Once()
	d.teamRepo.EXPECT().CountActiveTeams(mock.Anything).Return(1, nil).Once()

	uc := d.createUseCase()

	team, err := uc.Create(context.Background(), "TestTeam", captainID, false, false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrMaxTeamsReached))
	assert.Nil(t, team)
}
