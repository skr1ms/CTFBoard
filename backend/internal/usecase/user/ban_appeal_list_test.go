package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

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
