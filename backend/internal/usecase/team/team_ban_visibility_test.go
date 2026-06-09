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
)

func TestTeamUseCase_BanTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	team := &domain.Team{ID: teamID, Name: "Team"}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{}, nil).Twice()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().Ban(mock.Anything, teamID, "reason").Return(nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return([]*domain.SolveWithDetails{}, nil).Once()
	d.solveRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()

	actorID := uuid.New()

	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *domain.TeamAuditLog) bool {
		return l.TeamID == teamID && l.UserID != nil && *l.UserID == actorID && l.Action == domain.TeamActionBanned && l.Details["reason"] == "reason"
	})).Return(nil).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{}, nil).Once()

	uc := d.createUseCase()

	err := uc.BanTeam(context.Background(), teamID, "reason", false, actorID)

	assert.NoError(t, err)
}

func TestTeamUseCase_AfterTeamBanCommit_RevokesOnlyActuallyBannedUsers(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	adminID := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	uc := d.createUseCase()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{}, nil).Once()
	uc.afterTeamBanCommit(context.Background(), []uuid.UUID{teamID}, teamBanTxResult{
		memberIDs:     []uuid.UUID{adminID, userID},
		bannedUserIDs: []uuid.UUID{userID},
	}, true)

	assert.Equal(t, []uuid.UUID{userID}, d.jwtRevoker.revoked)
}

func TestTeamUseCase_BanTeam_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	userID := uuid.New()
	members := []*domain.User{{ID: userID}}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Twice()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, apperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	err := uc.BanTeam(context.Background(), teamID, "reason", false, uuid.Nil)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
}

func TestTeamUseCase_BanTeam_BanMembersRecordsOnlyNewNonAdminBans(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	adminID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	alreadyBannedID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	members := []*domain.User{
		{ID: adminID, Role: domain.RoleAdmin},
		{ID: alreadyBannedID, Role: domain.RoleUser, IsBanned: true},
		{ID: userID, Role: domain.RoleUser},
	}
	team := &domain.Team{ID: teamID, Name: "Team"}

	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return(members, nil).Twice()
	d.userRepo.EXPECT().Lock(mock.Anything, adminID).Return(nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, alreadyBannedID).Return(nil).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().Ban(mock.Anything, teamID, "reason").Return(nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return([]*domain.SolveWithDetails{}, nil).Once()
	d.solveRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().SoftBanByTeamID(mock.Anything, teamID).Return(nil).Once()
	d.userRepo.EXPECT().Ban(mock.Anything, userID, "reason").Return(nil).Once()
	d.userRepo.EXPECT().UpdateTeamIDBatch(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
		return assert.ElementsMatch(t, []uuid.UUID{adminID, alreadyBannedID, userID}, ids)
	}), (*uuid.UUID)(nil)).Return(nil).Once()
	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *domain.TeamAuditLog) bool {
		bannedIDs := parseUUIDSliceFromDetails(l.Details, "banned_user_ids")

		return l.TeamID == teamID &&
			l.Action == domain.TeamActionBanned &&
			assert.ObjectsAreEqual([]uuid.UUID{userID}, bannedIDs)
	})).Return(nil).Once()

	uc := d.createUseCase()
	result, err := uc.banTeamTx(context.Background(), teamID, "reason", true, uuid.New())

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{userID}, result.bannedUserIDs)
}

func TestTeamUseCase_UnbanTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	team := &domain.Team{ID: teamID, Name: "Team", IsBanned: true}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().GetLatestAuditLogByTeamIDAndAction(mock.Anything, teamID, "banned").Return(nil, nil).Once()
	d.userRepo.EXPECT().FilterIDsByTeamIDNullAndNotBanned(mock.Anything, mock.Anything).Return([]uuid.UUID(nil), nil).Twice()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{MaxTeamSize: 10}, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().Unban(mock.Anything, teamID).Return(nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{}, nil).Once()
	d.solveRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return([]*domain.SolveWithDetails{}, nil).Once()
	d.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, true).Return(nil).Once()

	actorID := uuid.New()

	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *domain.TeamAuditLog) bool {
		return l.TeamID == teamID && l.UserID != nil && *l.UserID == actorID && l.Action == domain.TeamActionUnbanned
	})).Return(nil).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{}, nil).Once()

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
	team := &domain.Team{ID: teamID, Name: "Team", CaptainID: userA, IsBanned: true}
	banLog := &domain.TeamAuditLog{
		TeamID:  teamID,
		Action:  domain.TeamActionBanned,
		Details: map[string]any{"reason": "cheat", "member_ids": []string{userA.String(), userB.String()}, "ban_members": true, "banned_user_ids": []string{userA.String()}},
	}
	userAModel := &domain.User{ID: userA, IsBanned: true}

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
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{MaxTeamSize: 10}, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().Unban(mock.Anything, teamID).Return(nil).Once()
	d.userRepo.EXPECT().GetByTeamID(mock.Anything, teamID).Return([]*domain.User{}, nil).Once()
	d.solveRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.submissionRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.awardRepo.EXPECT().RestoreByBannedTeamID(mock.Anything, teamID).Return(nil).Once()
	d.solveRepo.EXPECT().GetByTeamIDWithDetails(mock.Anything, teamID).Return([]*domain.SolveWithDetails{}, nil).Once()
	d.userRepo.EXPECT().FilterIDsByTeamIDNullAndNotBanned(mock.Anything, memberIDsMatcher).Return([]uuid.UUID{userA}, nil).Once()
	d.userRepo.EXPECT().UpdateTeamIDBatch(mock.Anything, []uuid.UUID{userA}, &teamID).Return(nil).Once()
	d.userRepo.EXPECT().SetWasInBannedTeamByIDs(mock.Anything, memberIDsMatcher, false).Return(nil).Once()

	actorID := uuid.New()

	d.teamRepo.EXPECT().CreateAuditLog(mock.Anything, mock.MatchedBy(func(l *domain.TeamAuditLog) bool {
		return l.TeamID == teamID && l.Action == domain.TeamActionUnbanned
	})).Return(nil).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{}, nil).Once()

	uc := d.createUseCase()

	err := uc.UnbanTeam(context.Background(), teamID, actorID)

	assert.NoError(t, err)
}

func TestTeamUseCase_UnbanTeam_DoesNotUseTimestampFallbackWithoutBannedUserIDs(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	userID := uuid.New()
	memberIDs := []uuid.UUID{userID}
	banLog := &domain.TeamAuditLog{
		TeamID: teamID,
		Action: domain.TeamActionBanned,
		Details: map[string]any{
			"ban_members": true,
			"member_ids":  []string{userID.String()},
		},
	}

	uc := d.createUseCase()
	err := uc.unbanTeamMembersByLog(context.Background(), banLog, &memberIDs)

	require.NoError(t, err)
}

func TestTeamUseCase_SetHidden_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	team := &domain.Team{ID: teamID, Name: "Team"}

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

func TestTeamUseCase_SetHiddenBulk_DedupesAndReportsAffected(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	firstID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	for _, teamID := range []uuid.UUID{firstID, secondID} {
		team := &domain.Team{ID: teamID}
		d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
		d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
		d.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, true).Return(nil).Once()
	}

	uc := d.createUseCase()
	result, err := uc.SetHiddenBulk(context.Background(), []uuid.UUID{secondID, firstID, secondID}, true)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.AffectedCount)
}

func TestTeamUseCase_SetHidden_Error(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, apperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	err := uc.SetHidden(context.Background(), teamID, true)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
}

func TestTeamUseCase_SetBracket_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	teamID := uuid.New()
	bracketID := uuid.New()
	team := &domain.Team{ID: teamID, Name: "Team"}

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
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(nil, apperr.ErrTeamNotFound).Once()

	uc := d.createUseCase()

	err := uc.SetBracket(context.Background(), teamID, nil)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamNotFound)
}
