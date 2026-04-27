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
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
)

type appealTestDeps struct {
	repo     *userMock.MockBanAppealRepository
	userRepo *userMock.MockUserRepository
	tm       *userMock.MockTransactionManager
}

func newAppealTestDeps(t *testing.T) *appealTestDeps {
	t.Helper()

	return &appealTestDeps{
		repo:     userMock.NewMockBanAppealRepository(t),
		userRepo: userMock.NewMockUserRepository(t),
		tm:       userMock.NewMockTransactionManager(t),
	}
}

func (d *appealTestDeps) createUseCase() *BanAppealUseCase {
	return NewBanAppealUseCase(d.repo, d.userRepo, d.tm)
}

func (d *appealTestDeps) setupTxRun() {
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Maybe()
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

// CreateAppeal tests

func TestBanAppealUseCase_CreateAppeal_NoPriorAppeal(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()

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
	boundary := newTestAppeal(userID, time.Now().Add(-(appealCooldown + time.Minute)), domain.AppealDecisionPending)

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

	d.repo.EXPECT().GetLatestByUserID(mock.Anything, userID).Return(nil, repoErr).Once()

	got, err := uc.CreateAppeal(context.Background(), userID, "message")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, repoErr)
}

// GetAppealsByUser tests

func TestBanAppealUseCase_GetAppealsByUser_HappyPath(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	appeals := []*domain.BanAppeal{
		newTestAppeal(userID, time.Now().Add(-2*24*time.Hour), domain.AppealDecisionPending),
		newTestAppeal(userID, time.Now().Add(-10*24*time.Hour), domain.AppealDecisionRejected),
	}

	d.repo.EXPECT().GetByUserID(mock.Anything, userID).Return(appeals, nil).Once()

	got, err := uc.GetAppealsByUser(context.Background(), userID)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestBanAppealUseCase_GetAppealsByUser_Empty(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()

	d.repo.EXPECT().GetByUserID(mock.Anything, userID).Return([]*domain.BanAppeal{}, nil).Once()

	got, err := uc.GetAppealsByUser(context.Background(), userID)
	assert.NoError(t, err)
	assert.Empty(t, got)
}

// ListAppeals tests

func TestBanAppealUseCase_ListAppeals_DefaultPagination(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	d.repo.EXPECT().List(mock.Anything, (*domain.AppealDecision)(nil), usecase.DefaultPerPage, 0).
		Return([]*domain.BanAppeal{}, int64(0), nil).Once()

	result, err := uc.ListAppeals(context.Background(), nil, 1, 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, usecase.DefaultPerPage, result.PerPage)
}

func TestBanAppealUseCase_ListAppeals_WithDecisionFilter(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	dec := domain.AppealDecisionPending
	userID := uuid.New()
	appeals := []*domain.BanAppeal{
		newTestAppeal(userID, time.Now(), dec),
	}

	d.repo.EXPECT().List(mock.Anything, &dec, 10, 0).
		Return(appeals, int64(1), nil).Once()

	result, err := uc.ListAppeals(context.Background(), &dec, 1, 10)
	assert.NoError(t, err)
	assert.Len(t, result.Data, 1)
	assert.Equal(t, int64(1), result.Total)
}

func TestBanAppealUseCase_ListAppeals_PageZeroClamped(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	d.repo.EXPECT().List(mock.Anything, (*domain.AppealDecision)(nil), usecase.DefaultPerPage, 0).
		Return([]*domain.BanAppeal{}, int64(0), nil).Once()

	result, err := uc.ListAppeals(context.Background(), nil, 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Page)
}

func TestBanAppealUseCase_ListAppeals_PerPageTooLargeClamped(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	d.repo.EXPECT().List(mock.Anything, (*domain.AppealDecision)(nil), usecase.DefaultPerPage, 0).
		Return([]*domain.BanAppeal{}, int64(0), nil).Once()

	result, err := uc.ListAppeals(context.Background(), nil, 1, usecase.DefaultMaxPerPage+1)
	assert.NoError(t, err)
	assert.Equal(t, usecase.DefaultPerPage, result.PerPage)
}

func TestBanAppealUseCase_ListAppeals_Pagination_Page2(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	perPage := 5
	expectedOffset := 5 // (page=2-1) * perPage=5

	d.repo.EXPECT().List(mock.Anything, (*domain.AppealDecision)(nil), perPage, expectedOffset).
		Return([]*domain.BanAppeal{}, int64(10), nil).Once()

	result, err := uc.ListAppeals(context.Background(), nil, 2, perPage)
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, int64(10), result.Total)
}

// ReviewAppeal tests

func TestBanAppealUseCase_ReviewAppeal_Resolved_UnbansUser(t *testing.T) {
	t.Parallel()
	d := newAppealTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	appealID := uuid.New()
	actorID := uuid.New()
	adminResp := "appeal granted"
	appeal := newTestAppeal(userID, time.Now().Add(-24*time.Hour), domain.AppealDecisionPending)
	appeal.ID = appealID

	d.repo.EXPECT().GetByID(mock.Anything, appealID).Return(appeal, nil).Once()
	d.setupTxRun()
	d.repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(a *domain.BanAppeal) bool {
		return a.ID == appealID && a.Decision == domain.AppealDecisionResolved && a.AdminResponse == &adminResp
	})).Return(nil).Once()
	d.userRepo.EXPECT().Unban(mock.Anything, userID).Return(nil).Once()

	got, err := uc.ReviewAppeal(context.Background(), appealID, domain.AppealDecisionResolved, &adminResp, actorID)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, domain.AppealDecisionResolved, got.Decision)
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
	d.repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(a *domain.BanAppeal) bool {
		return a.ID == appealID && a.Decision == domain.AppealDecisionRejected
	})).Return(nil).Once()
	// Unban must NOT be called - ensured by mock expecting exactly 0 calls

	got, err := uc.ReviewAppeal(context.Background(), appealID, domain.AppealDecisionRejected, &adminResp, actorID)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, domain.AppealDecisionRejected, got.Decision)
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

	d.repo.EXPECT().GetByID(mock.Anything, appealID).Return(appeal, nil).Once()
	d.setupTxRun()
	d.repo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Once()
	d.userRepo.EXPECT().Unban(mock.Anything, userID).Return(unbanErr).Once()

	got, err := uc.ReviewAppeal(context.Background(), appealID, domain.AppealDecisionResolved, nil, uuid.New())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, unbanErr)
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

// CanAppeal tests

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
