package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
)

type trackingTestDeps struct {
	trackingRepo *userMock.MockTrackingRepository
}

func newTrackingTestDeps(t *testing.T) *trackingTestDeps {
	t.Helper()

	return &trackingTestDeps{trackingRepo: userMock.NewMockTrackingRepository(t)}
}

func (d *trackingTestDeps) createUseCase() *TrackingUseCase {
	return NewTrackingUseCase(TrackingDeps{TrackingRepo: d.trackingRepo})
}

func TestTrackingUseCase_Track_Success(t *testing.T) {
	t.Parallel()
	d := newTrackingTestDeps(t)
	d.trackingRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.TrackingEntry")).Return(nil)

	err := d.createUseCase().Track(context.Background(), uuid.New(), "127.0.0.1", "Mozilla/5.0")
	require.NoError(t, err)
}

func TestTrackingUseCase_Track_RepoError(t *testing.T) {
	t.Parallel()
	d := newTrackingTestDeps(t)
	d.trackingRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.TrackingEntry")).Return(errors.New("db error"))

	err := d.createUseCase().Track(context.Background(), uuid.New(), "127.0.0.1", "Mozilla/5.0")
	assert.Error(t, err)
}

func TestTrackingUseCase_GetByUser_Success(t *testing.T) {
	t.Parallel()
	d := newTrackingTestDeps(t)
	userID := uuid.New()
	entries := []*domain.TrackingEntry{
		{ID: uuid.New(), UserID: userID, IP: "10.0.0.1"},
	}
	d.trackingRepo.On("GetByUser", mock.Anything, userID, 10, 0).Return(entries, nil)
	d.trackingRepo.On("CountByUser", mock.Anything, userID).Return(1, nil)

	got, err := d.createUseCase().GetByUser(context.Background(), userID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), got.Total)
	assert.Len(t, got.Data, 1)
	assert.Equal(t, userID, got.Data[0].UserID)
}

func TestTrackingUseCase_GetByUser_NotFound(t *testing.T) {
	t.Parallel()
	d := newTrackingTestDeps(t)
	userID := uuid.New()
	d.trackingRepo.On("GetByUser", mock.Anything, userID, mock.Anything, mock.Anything).Return(([]*domain.TrackingEntry)(nil), errors.New("not found"))

	got, err := d.createUseCase().GetByUser(context.Background(), userID, 1, 10)
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTrackingUseCase_TrackChallengeOpen_Success(t *testing.T) {
	t.Parallel()
	d := newTrackingTestDeps(t)
	d.trackingRepo.On("CreateChallengeOpen", mock.Anything, mock.AnythingOfType("*domain.ChallengeOpen")).Return(nil)

	err := d.createUseCase().TrackChallengeOpen(context.Background(), uuid.New(), uuid.New(), "192.168.1.1")
	require.NoError(t, err)
}

func TestTrackingUseCase_TrackChallengeOpen_RepoError(t *testing.T) {
	t.Parallel()
	d := newTrackingTestDeps(t)
	d.trackingRepo.On("CreateChallengeOpen", mock.Anything, mock.AnythingOfType("*domain.ChallengeOpen")).Return(errors.New("db error"))

	err := d.createUseCase().TrackChallengeOpen(context.Background(), uuid.New(), uuid.New(), "192.168.1.1")
	assert.Error(t, err)
}
