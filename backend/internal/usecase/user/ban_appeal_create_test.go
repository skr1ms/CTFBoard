package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestBanAppealUseCase_CreateAppeal_NoPriorAppeal(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()

	d.setupCreateAppealTx(userID)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, IsBanned: true}, nil).Once()
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(nil, nil).Once()
	d.repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(a *domain.BanAppeal) bool {
		return a.UserID == userID && a.Message == "please unban" && a.Decision == domain.AppealDecisionPending
	})).Return(nil).Once()

	got, err := uc.CreateAppeal(context.Background(), userID, "please unban")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, domain.AppealDecisionPending, got.Decision)
}

func TestBanAppealUseCase_CreateAppeal_WithinCooldown(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	recent := newTestAppeal(userID, time.Now().Add(-1*time.Hour), domain.AppealDecisionPending)

	d.setupCreateAppealTx(userID)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, IsBanned: true}, nil).Once()
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(recent, nil).Once()

	got, err := uc.CreateAppeal(context.Background(), userID, "please unban")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, apperr.ErrAppealRateLimited)
}

func TestBanAppealUseCase_CreateAppeal_AfterCooldown(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	old := newTestAppeal(userID, time.Now().Add(-8*24*time.Hour), domain.AppealDecisionRejected)

	d.setupCreateAppealTx(userID)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, IsBanned: true}, nil).Once()
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(old, nil).Once()
	d.repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(a *domain.BanAppeal) bool {
		return a.UserID == userID
	})).Return(nil).Once()

	got, err := uc.CreateAppeal(context.Background(), userID, "please unban again")
	assert.NoError(t, err)
	assert.NotNil(t, got)
}

func TestBanAppealUseCase_CreateAppeal_ExactlyCooldownBoundary(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	// 7 days + 1 minute ago - just past cooldown
	boundary := newTestAppeal(userID, time.Now().Add(-(appealCooldown + time.Minute)), domain.AppealDecisionRejected)

	d.setupCreateAppealTx(userID)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, IsBanned: true}, nil).Once()
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(boundary, nil).Once()
	d.repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()

	got, err := uc.CreateAppeal(context.Background(), userID, "message")
	assert.NoError(t, err)
	assert.NotNil(t, got)
}

func TestBanAppealUseCase_CreateAppeal_RepoError(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	repoErr := errors.New("db error")

	d.setupCreateAppealTx(userID)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, IsBanned: true}, nil).Once()
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(nil, repoErr).Once()

	got, err := uc.CreateAppeal(context.Background(), userID, "message")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, repoErr)
}

func TestBanAppealUseCase_CreateAppeal_NotBannedRejected(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()

	d.setupCreateAppealTx(userID)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, IsBanned: false}, nil).Once()

	got, err := uc.CreateAppeal(context.Background(), userID, "please unban")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, apperr.ErrAccessDenied)
}

func TestBanAppealUseCase_CreateAppeal_WasInBannedTeamAllowed(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()

	d.setupCreateAppealTx(userID)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{
		ID:              userID,
		IsBanned:        false,
		WasInBannedTeam: true,
	}, nil).Once()
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(nil, nil).Once()
	d.repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(a *domain.BanAppeal) bool {
		return a.UserID == userID && a.Message == "please unban"
	})).Return(nil).Once()

	got, err := uc.CreateAppeal(context.Background(), userID, "please unban")
	assert.NoError(t, err)
	assert.NotNil(t, got)
}
