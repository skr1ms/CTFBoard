package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
)

type appealTestDeps struct {
	repo        *userMock.MockBanAppealRepository
	userRepo    *userMock.MockUserRepository
	tm          *userMock.MockTransactionManager
	banRestorer *appealBanRestorer
}

func newAppealTestDeps(t *testing.T) *appealTestDeps {
	t.Helper()

	return &appealTestDeps{
		repo:        userMock.NewMockBanAppealRepository(t),
		userRepo:    userMock.NewMockUserRepository(t),
		tm:          userMock.NewMockTransactionManager(t),
		banRestorer: &appealBanRestorer{},
	}
}

func (d *appealTestDeps) createUseCase() *BanAppealUseCase {
	return NewBanAppealUseCase(d.repo, d.userRepo, d.tm, d.banRestorer)
}

func (d *appealTestDeps) setupTxRun() {
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Maybe()
}

func (d *appealTestDeps) setupCreateAppealTx(userID uuid.UUID) {
	d.setupTxRun()
	d.userRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, banAppealAdvisoryKey(userID)).Return(nil).Once()
}

func newTestAppeal(userID uuid.UUID, createdAt time.Time, decision domain.AppealDecision) *domain.BanAppeal {
	return &domain.BanAppeal{
		ID:        uuid.New(),
		UserID:    userID,
		Message:   "please unban me",
		Decision:  decision,
		CreatedAt: createdAt,
	}
}

type appealBanRestorer struct {
	restoreErr             error
	restoreCalls           int
	invalidatedUserID      uuid.UUID
	invalidatedTeamIDs     []uuid.UUID
	teamIDsToInvalidate    []uuid.UUID
	invalidatedAfterCommit bool
}

func (r *appealBanRestorer) restoreAppealedUserBanTx(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	r.restoreCalls++

	return r.teamIDsToInvalidate, r.restoreErr
}

func (r *appealBanRestorer) invalidateAppealedUserBanRestore(_ context.Context, userID uuid.UUID, teamIDs []uuid.UUID) {
	r.invalidatedUserID = userID
	r.invalidatedTeamIDs = teamIDs
	r.invalidatedAfterCommit = true
}
