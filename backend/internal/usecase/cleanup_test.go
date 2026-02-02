package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/internal/usecase/team/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCleanupUseCase_CleanupDeletedTeams_Success(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	uc := NewCleanupUseCase(teamRepo)
	ctx := context.Background()
	olderThan := 24 * time.Hour

	teamRepo.EXPECT().
		HardDeleteTeams(ctx, mock.MatchedBy(func(t time.Time) bool { return !t.IsZero() })).
		Return(nil).Once()

	err := uc.CleanupDeletedTeams(ctx, olderThan)
	assert.NoError(t, err)
	teamRepo.AssertExpectations(t)
}

func TestCleanupUseCase_CleanupDeletedTeams_Error(t *testing.T) {
	teamRepo := mocks.NewMockTeamRepository(t)
	uc := NewCleanupUseCase(teamRepo)
	ctx := context.Background()
	olderThan := 24 * time.Hour
	expectedErr := errors.New("db error")

	teamRepo.EXPECT().
		HardDeleteTeams(ctx, mock.MatchedBy(func(t time.Time) bool { return !t.IsZero() })).
		Return(expectedErr).Once()

	err := uc.CleanupDeletedTeams(ctx, olderThan)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CleanupUseCase")
	assert.Contains(t, err.Error(), expectedErr.Error())
	teamRepo.AssertExpectations(t)
}
