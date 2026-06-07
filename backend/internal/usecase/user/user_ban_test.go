package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
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
	d.solveRepo.EXPECT().GetByUserIDWithDetails(mock.Anything, userID).Return([]*domain.SolveWithDetails{}, nil).Once()
	d.jwtService.EXPECT().RevokeAllForUser(mock.Anything, userID).Return(nil).Once()

	uc := d.createUseCase()
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
	d.solveRepo.EXPECT().GetByUserIDWithDetails(mock.Anything, userID).Return([]*domain.SolveWithDetails{}, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, true).Return(nil).Once()
	d.jwtService.EXPECT().RevokeAllForUser(mock.Anything, userID).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.BanUser(context.Background(), userID, "reason", actorID)

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
	d.userRepo.EXPECT().SetWasInBannedTeamByIDs(mock.Anything, []uuid.UUID{userID}, false).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, false).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.UnbanUser(context.Background(), userID, actorID)

	assert.NoError(t, err)
}
