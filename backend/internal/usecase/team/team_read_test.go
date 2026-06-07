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

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestTeamUseCase_GetByID_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	expectedTeam := &domain.Team{
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

	user := &domain.User{
		ID:     userID,
		TeamID: &teamID,
	}

	team := &domain.Team{
		ID:          teamID,
		Name:        "MyTeam",
		InviteToken: uuid.New(),
		CaptainID:   userID,
	}

	members := []*domain.User{user}

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{MinTeamSize: 0}, nil).Once()

	uc := d.createUseCase()

	result, err := uc.GetMyTeam(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, teamID, result.Team.ID)
	assert.Equal(t, "MyTeam", result.Team.Name)
	assert.NotNil(t, result.Members)
	assert.Len(t, result.Members, 1)
	assert.Equal(t, 0, result.MinTeamSize)
	assert.True(t, result.MeetsMinSize)
}

func TestTeamUseCase_GetTeamMembers_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	members := []*domain.User{
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

func TestTeamUseCase_GetByID_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, apperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	team, err := uc.GetByID(context.Background(), teamID)

	assert.Error(t, err)
	assert.Nil(t, team)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
}

func TestTeamUseCase_GetMyTeam_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	userID := uuid.New()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, apperr.ErrUserNotFound).Once()

	uc := d.createUseCase()

	team, err := uc.GetMyTeam(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, team)
	assert.ErrorIs(t, err, apperr.ErrUserNotFound)
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

func TestTeamUseCase_ListTeams_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	d.tm.EXPECT().ReadOnly(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	teams := []*domain.Team{{ID: uuid.New(), Name: "Team1"}}
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

func TestTeamUseCase_GetTeamSolves_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	solves := []*domain.SolveWithDetails{
		{
			Solve:          domain.Solve{ChallengeID: uuid.New(), SolvedAt: time.Now()},
			ChallengeTitle: "Ch1",
		},
	}

	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID}, nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return(solves, nil).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{}, nil).Once()

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
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, apperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamSolves(context.Background(), teamID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
	assert.Nil(t, result)
}

func TestTeamUseCase_GetTeamFails_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()

	var fails []*domain.SubmissionWithDetails

	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID}, nil).Once()
	d.submissionRepo.EXPECT().GetFailsByTeam(mock.Anything, teamID, 10, 0).Return(fails, nil).Once()
	d.submissionRepo.EXPECT().CountFailsByTeam(mock.Anything, teamID).Return(int64(0), nil).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamFails(context.Background(), teamID, 1, 10)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Data)
	assert.Equal(t, int64(0), result.Total)
}

func TestTeamUseCase_GetTeamFails_Error_TeamNotFound(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, apperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamFails(context.Background(), teamID, 1, 10)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
	assert.Nil(t, result)
}

func TestTeamUseCase_GetTeamAwards_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	awards := []*domain.Award{{ID: uuid.New(), TeamID: teamID, Value: 50}}

	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID}, nil).Once()
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
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, apperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	result, err := uc.GetTeamAwards(context.Background(), teamID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
	assert.Nil(t, result)
}
