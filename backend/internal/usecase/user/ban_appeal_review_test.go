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

func TestBanAppealUseCase_ReviewAppeal_Resolved_UnbansUser(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	appealID := uuid.New()
	actorID := uuid.New()
	teamID := uuid.New()
	adminResp := "appeal granted"
	appeal := newTestAppeal(userID, time.Now().Add(-24*time.Hour), domain.AppealDecisionPending)
	appeal.ID = appealID
	d.banRestorer.teamIDsToInvalidate = []uuid.UUID{teamID}

	d.repo.EXPECT().GetByID(mock.Anything, appealID).Return(appeal, nil).Once()
	d.setupTxRun()
	d.repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(a *domain.BanAppeal) bool {
		return a.ID == appealID && a.Decision == domain.AppealDecisionResolved && a.AdminResponse == &adminResp
	})).Return(nil).Once()

	got, err := uc.ReviewAppeal(context.Background(), appealID, domain.AppealDecisionResolved, &adminResp, actorID)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, domain.AppealDecisionResolved, got.Decision)
	assert.Equal(t, 1, d.banRestorer.restoreCalls)
	assert.Equal(t, userID, d.banRestorer.invalidatedUserID)
	assert.Equal(t, []uuid.UUID{teamID}, d.banRestorer.invalidatedTeamIDs)
	assert.True(t, d.banRestorer.invalidatedAfterCommit)
}

func TestBanAppealUseCase_ReviewAppeal_Rejected_NoUnban(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	appealID := uuid.New()
	actorID := uuid.New()
	adminResp := "appeal denied"
	appeal := newTestAppeal(userID, time.Now().Add(-24*time.Hour), domain.AppealDecisionPending)
	appeal.ID = appealID

	d.repo.EXPECT().GetByID(mock.Anything, appealID).Return(appeal, nil).Once()
	d.setupTxRun()
	d.repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(a *domain.BanAppeal) bool {
		return a.ID == appealID && a.Decision == domain.AppealDecisionRejected
	})).Return(nil).Once()
	// Unban must NOT be called - ensured by mock expecting exactly 0 calls

	got, err := uc.ReviewAppeal(context.Background(), appealID, domain.AppealDecisionRejected, &adminResp, actorID)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, domain.AppealDecisionRejected, got.Decision)
	assert.Equal(t, 0, d.banRestorer.restoreCalls)
}

func TestBanAppealUseCase_ReviewAppeal_AlreadyReviewed(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	appealID := uuid.New()
	// Already resolved - not pending
	appeal := newTestAppeal(userID, time.Now().Add(-48*time.Hour), domain.AppealDecisionResolved)
	appeal.ID = appealID

	d.repo.EXPECT().GetByID(mock.Anything, appealID).Return(appeal, nil).Once()

	got, err := uc.ReviewAppeal(context.Background(), appealID, domain.AppealDecisionResolved, nil, uuid.New())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, apperr.ErrAccessDenied)
}

func TestBanAppealUseCase_ReviewAppeal_NotFound(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	appealID := uuid.New()

	d.repo.EXPECT().GetByID(mock.Anything, appealID).Return(nil, apperr.ErrAppealNotFound).Once()

	got, err := uc.ReviewAppeal(context.Background(), appealID, domain.AppealDecisionResolved, nil, uuid.New())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, apperr.ErrAppealNotFound)
}

func TestBanAppealUseCase_ReviewAppeal_TxError(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	appealID := uuid.New()
	txErr := errors.New("tx failed")
	appeal := newTestAppeal(userID, time.Now().Add(-24*time.Hour), domain.AppealDecisionPending)
	appeal.ID = appealID

	d.repo.EXPECT().GetByID(mock.Anything, appealID).Return(appeal, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).Return(txErr).Once()

	got, err := uc.ReviewAppeal(context.Background(), appealID, domain.AppealDecisionResolved, nil, uuid.New())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, txErr)
}

func TestBanAppealUseCase_ReviewAppeal_Resolved_UnbanError(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	appealID := uuid.New()
	unbanErr := errors.New("unban failed")
	appeal := newTestAppeal(userID, time.Now().Add(-24*time.Hour), domain.AppealDecisionPending)
	appeal.ID = appealID
	d.banRestorer.restoreErr = unbanErr

	d.repo.EXPECT().GetByID(mock.Anything, appealID).Return(appeal, nil).Once()
	d.setupTxRun()

	got, err := uc.ReviewAppeal(context.Background(), appealID, domain.AppealDecisionResolved, nil, uuid.New())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, unbanErr)
	assert.Equal(t, 1, d.banRestorer.restoreCalls)
	assert.False(t, d.banRestorer.invalidatedAfterCommit)
}

func TestBanAppealUseCase_ReviewAppeal_Rejected_AlreadyRejected(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	appealID := uuid.New()
	appeal := newTestAppeal(userID, time.Now().Add(-48*time.Hour), domain.AppealDecisionRejected)
	appeal.ID = appealID

	d.repo.EXPECT().GetByID(mock.Anything, appealID).Return(appeal, nil).Once()

	got, err := uc.ReviewAppeal(context.Background(), appealID, domain.AppealDecisionRejected, nil, uuid.New())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, apperr.ErrAccessDenied)
}
