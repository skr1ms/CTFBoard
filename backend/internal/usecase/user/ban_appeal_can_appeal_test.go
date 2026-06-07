package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestBanAppealUseCase_CanAppeal_NoHistory(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(nil, nil).Once()

	canAppeal, hasPending, err := uc.CanAppeal(context.Background(), userID)
	assert.NoError(t, err)
	assert.True(t, canAppeal)
	assert.False(t, hasPending)
}

func TestBanAppealUseCase_CanAppeal_PendingAppeal(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	pending := newTestAppeal(userID, time.Now().Add(-1*time.Hour), domain.AppealDecisionPending)
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(pending, nil).Once()

	canAppeal, hasPending, err := uc.CanAppeal(context.Background(), userID)
	assert.NoError(t, err)
	assert.False(t, canAppeal)
	assert.True(t, hasPending)
}

func TestBanAppealUseCase_CanAppeal_WithinCooldownAfterReject(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	recent := newTestAppeal(userID, time.Now().Add(-24*time.Hour), domain.AppealDecisionRejected)
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(recent, nil).Once()

	canAppeal, hasPending, err := uc.CanAppeal(context.Background(), userID)
	assert.NoError(t, err)
	assert.False(t, canAppeal)
	assert.False(t, hasPending)
}

func TestBanAppealUseCase_CanAppeal_AfterCooldown(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	old := newTestAppeal(userID, time.Now().Add(-8*24*time.Hour), domain.AppealDecisionRejected)
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(old, nil).Once()

	canAppeal, hasPending, err := uc.CanAppeal(context.Background(), userID)
	assert.NoError(t, err)
	assert.True(t, canAppeal)
	assert.False(t, hasPending)
}

func TestBanAppealUseCase_CanAppeal_RepoError(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	repoErr := errors.New("db down")
	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(nil, repoErr).Once()

	canAppeal, hasPending, err := uc.CanAppeal(context.Background(), userID)
	assert.ErrorIs(t, err, repoErr)
	assert.False(t, canAppeal)
	assert.False(t, hasPending)
}
