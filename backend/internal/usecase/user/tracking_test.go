package user

import (
	"context"
	"errors"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTrackingUseCase_Track_Success(t *testing.T) {
	t.Parallel()
	h := NewTrackingTestHelper(t)
	h.TrackingRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.TrackingEntry")).Return(nil)

	err := h.CreateUseCase().Track(context.Background(), uuid.New(), "127.0.0.1", "Mozilla/5.0")
	require.NoError(t, err)
}

func TestTrackingUseCase_Track_RepoError(t *testing.T) {
	t.Parallel()
	h := NewTrackingTestHelper(t)
	h.TrackingRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.TrackingEntry")).Return(errors.New("db error"))

	err := h.CreateUseCase().Track(context.Background(), uuid.New(), "127.0.0.1", "Mozilla/5.0")
	assert.Error(t, err)
}

func TestTrackingUseCase_GetByUser_Success(t *testing.T) {
	t.Parallel()
	h := NewTrackingTestHelper(t)
	userID := uuid.New()
	entries := []*entity.TrackingEntry{
		{ID: uuid.New(), UserID: userID, IP: "10.0.0.1"},
	}
	h.TrackingRepo.On("GetByUser", mock.Anything, userID, 10, 0).Return(entries, nil)
	h.TrackingRepo.On("CountByUser", mock.Anything, userID).Return(1, nil)

	got, err := h.CreateUseCase().GetByUser(context.Background(), userID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), got.Total)
	assert.Len(t, got.Data, 1)
	assert.Equal(t, userID, got.Data[0].UserID)
}

func TestTrackingUseCase_GetByUser_NotFound(t *testing.T) {
	t.Parallel()
	h := NewTrackingTestHelper(t)
	userID := uuid.New()
	h.TrackingRepo.On("GetByUser", mock.Anything, userID, mock.Anything, mock.Anything).Return(([]*entity.TrackingEntry)(nil), errors.New("not found"))

	got, err := h.CreateUseCase().GetByUser(context.Background(), userID, 1, 10)
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTrackingUseCase_TrackChallengeOpen_Success(t *testing.T) {
	t.Parallel()
	h := NewTrackingTestHelper(t)
	h.TrackingRepo.On("CreateChallengeOpen", mock.Anything, mock.AnythingOfType("*entity.ChallengeOpen")).Return(nil)

	err := h.CreateUseCase().TrackChallengeOpen(context.Background(), uuid.New(), uuid.New(), "192.168.1.1")
	require.NoError(t, err)
}

func TestTrackingUseCase_TrackChallengeOpen_RepoError(t *testing.T) {
	t.Parallel()
	h := NewTrackingTestHelper(t)
	h.TrackingRepo.On("CreateChallengeOpen", mock.Anything, mock.AnythingOfType("*entity.ChallengeOpen")).Return(errors.New("db error"))

	err := h.CreateUseCase().TrackChallengeOpen(context.Background(), uuid.New(), uuid.New(), "192.168.1.1")
	assert.Error(t, err)
}
