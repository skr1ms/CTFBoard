package team

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func TestTeamUseCase_AdminListTeams_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	d.tm.EXPECT().ReadOnly(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	teams := []*domain.Team{{ID: uuid.New(), Name: "AdminTeam1"}}
	filter := repo.TeamAdminSearchFilter{
		Search:     nil,
		BanStatus:  repo.TeamAdminBanStatusAll,
		Visibility: repo.TeamAdminVisibilityAll,
	}
	d.teamRepo.EXPECT().SearchAdmin(mock.Anything, filter, 10, 0).Return(teams, nil).Once()
	d.teamRepo.EXPECT().CountSearchAdmin(mock.Anything, filter).Return(int64(1), nil).Once()

	uc := d.createUseCase()

	result, err := uc.AdminListTeams(
		context.Background(),
		nil,
		usecase.AdminTeamBanStatusAll,
		usecase.AdminTeamVisibilityAll,
		1,
		10,
	)

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

	_, err := uc.AdminListTeams(
		context.Background(),
		nil,
		usecase.AdminTeamBanStatusAll,
		usecase.AdminTeamVisibilityAll,
		1,
		10,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AdminListTeams")
}

func TestTeamUseCase_AdminUpdate_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	newName := "UpdatedName"
	updatedTeam := &domain.Team{ID: teamID, Name: newName}

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
	members := []*domain.User{{ID: memberID, TeamID: &teamID}}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Twice()
	d.userRepo.EXPECT().Lock(mock.Anything, memberID).Return(nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return([]*domain.SolveWithDetails{}, nil).Once()
	d.solveRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().DeleteByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamIDBatch(mock.Anything, []uuid.UUID{memberID}, (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().Delete(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *domain.TeamAuditLog) bool {
		return l.TeamID == teamID && l.Action == domain.TeamActionDeleted && l.Details["reason"] == "deleted_by_admin"
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

	nonSoloTeam := &domain.Team{ID: teamID, IsSolo: false}
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nonSoloTeam, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{}, nil).Once()
	d.compRepo.EXPECT().GetForUpdate(mock.Anything).Return(&domain.Competition{Mode: "teams_only", MaxTeamSize: 5}, nil).Once()
	d.userRepo.EXPECT().UpdateTeamID(mock.Anything, userID, &teamID).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *domain.TeamAuditLog) bool {
		return l.TeamID == teamID && l.UserID != nil && *l.UserID == userID && l.Action == domain.TeamActionJoined
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

	nonSoloTeam := &domain.Team{ID: teamID, IsSolo: false}
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nonSoloTeam, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()

	uc := d.createUseCase()

	err := uc.AdminAddMember(context.Background(), teamID, userID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamConflict)
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
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *domain.TeamAuditLog) bool {
		return l.TeamID == teamID && l.UserID != nil && *l.UserID == memberID && l.Action == domain.TeamActionMemberKicked
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
	assert.ErrorIs(t, err, apperr.ErrCaptainCannotLeave)
}
