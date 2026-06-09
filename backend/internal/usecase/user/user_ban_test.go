package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	teamMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mock"
)

func TestUserUseCase_BanUser_Success_NoSoloTeam(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	actorID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser, TeamID: nil}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().Ban(mock.Anything, userID, "reason").Return(nil).Once()
	d.solveRepo.EXPECT().GetModerationAffectedSolvesByUserID(mock.Anything, userID).Return([]*domain.ModerationAffectedSolve{}, nil).Once()
	d.jwtService.EXPECT().RevokeAllForUser(mock.Anything, userID).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.BanUser(context.Background(), userID, "reason", actorID)

	assert.NoError(t, err)
}

func TestUserUseCase_BanUsers_DedupesAndReportsAffected(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	firstID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	actorID := uuid.New()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	for _, userID := range []uuid.UUID{firstID, secondID} {
		user := &domain.User{ID: userID, Role: domain.RoleUser}
		d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
		d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
		d.userRepo.EXPECT().Ban(mock.Anything, userID, "reason").Return(nil).Once()
		d.solveRepo.EXPECT().GetModerationAffectedSolvesByUserID(mock.Anything, userID).Return([]*domain.ModerationAffectedSolve{}, nil).Once()
		d.jwtService.EXPECT().RevokeAllForUser(mock.Anything, userID).Return(nil).Once()
	}

	uc := d.createUseCase()
	result, err := uc.BanUsers(context.Background(), []uuid.UUID{secondID, firstID, secondID}, "reason", actorID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.AffectedCount)
}

func TestUserUseCase_BanUser_RecalculatesHiddenOnlySolve(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	challengeRepo := teamMock.NewMockChallengeRepository(t)
	userID := uuid.New()
	actorID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().Ban(mock.Anything, userID, "reason").Return(nil).Once()
	d.solveRepo.EXPECT().GetModerationAffectedSolvesByUserID(mock.Anything, userID).Return([]*domain.ModerationAffectedSolve{
		{TeamID: teamID, ChallengeID: challengeID},
	}, nil).Once()
	d.solveRepo.EXPECT().SoftBanByTeamIDAndUserID(mock.Anything, teamID, userID).Return(nil).Once()
	challengeRepo.EXPECT().RecalculateSolveCounts(mock.Anything, []uuid.UUID{challengeID}).Return(nil).Once()
	challengeRepo.EXPECT().GetByIDs(mock.Anything, []uuid.UUID{challengeID}).Return(map[uuid.UUID]*domain.Challenge{
		challengeID: {ID: challengeID, Points: 100},
	}, nil).Once()
	d.jwtService.EXPECT().RevokeAllForUser(mock.Anything, userID).Return(nil).Once()

	uc := NewUserUseCase(UserDeps{
		UserRepo: d.userRepo, TeamRepo: d.teamRepo, SolveRepo: d.solveRepo, ChallengeRepo: challengeRepo,
		TM: d.tm, JWTService: d.jwtService, APITokenRevoker: d.apiTokenRevoker,
	})

	err := uc.BanUser(context.Background(), userID, "reason", actorID)

	assert.NoError(t, err)
}

func TestUserUseCase_BanUser_Success_HidesSoloTeamInTx(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	actorID := uuid.New()
	teamID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser, TeamID: &teamID}
	team := &domain.Team{ID: teamID, IsSolo: true, IsHidden: false}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().Ban(mock.Anything, userID, "reason").Return(nil).Once()
	d.solveRepo.EXPECT().GetModerationAffectedSolvesByUserID(mock.Anything, userID).Return([]*domain.ModerationAffectedSolve{}, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, true).Return(nil).Once()
	d.jwtService.EXPECT().RevokeAllForUser(mock.Anything, userID).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.BanUser(context.Background(), userID, "reason", actorID)

	assert.NoError(t, err)
}

func TestUserUseCase_UnbanUsers_RejectsInheritedTeamBan(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	actorID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser, WasInBannedTeam: true}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()

	uc := d.createUseCase()
	result, err := uc.UnbanUsers(context.Background(), []uuid.UUID{userID}, actorID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "team-inherited ban must be cleared by unbanning the team")
}

func TestUserUseCase_UnbanUser_RejectsInheritedTeamBan(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	actorID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser, WasInBannedTeam: true}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()

	uc := d.createUseCase()
	err := uc.UnbanUser(context.Background(), userID, actorID)

	require.Error(t, err)
	assert.ErrorContains(t, err, "team-inherited ban must be cleared by unbanning the team")
}

func TestUserUseCase_RestoreAppealedUserBanTx_RejectsInheritedTeamBan(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser, WasInBannedTeam: true}

	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()

	uc := d.createUseCase()
	teamIDs, err := uc.restoreAppealedUserBanTx(context.Background(), userID)

	require.Error(t, err)
	assert.Nil(t, teamIDs)
	assert.ErrorContains(t, err, "team-inherited ban must be cleared by unbanning the team")
}

func TestUserUseCase_UnbanUser_DirectBanDoesNotClearInheritedTeamBan(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	actorID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser, IsBanned: true, WasInBannedTeam: true}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().Unban(mock.Anything, userID).Return(nil).Once()
	uc := d.createUseCase()
	err := uc.UnbanUser(context.Background(), userID, actorID)

	assert.NoError(t, err)
}

func TestUserUseCase_UnbanUser_Success_ShowsSoloTeamInTx(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	actorID := uuid.New()
	teamID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser, TeamID: &teamID, IsBanned: true}
	team := &domain.Team{ID: teamID, IsSolo: true, IsHidden: true}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().Unban(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, false).Return(nil).Once()
	uc := d.createUseCase()
	err := uc.UnbanUser(context.Background(), userID, actorID)

	assert.NoError(t, err)
}
