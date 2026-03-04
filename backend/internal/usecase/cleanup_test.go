package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCleanupUseCase_CleanupDeletedTeams_Success(t *testing.T) {
	t.Parallel()
	h := NewCleanupTestHelper(t)
	ctx := context.Background()

	h.TeamRepo.EXPECT().
		HardDeleteTeams(ctx, mock.MatchedBy(func(t interface{ IsZero() bool }) bool { return !t.IsZero() })).
		Return(nil).Once()

	err := h.CreateUseCase().CleanupDeletedTeams(ctx, h.DefaultOlderThan())
	assert.NoError(t, err)
	h.TeamRepo.AssertExpectations(t)
}

func TestCleanupUseCase_CleanupDeletedTeams_Error(t *testing.T) {
	t.Parallel()
	h := NewCleanupTestHelper(t)
	ctx := context.Background()
	expectedErr := errors.New("db error")

	h.TeamRepo.EXPECT().
		HardDeleteTeams(ctx, mock.MatchedBy(func(t interface{ IsZero() bool }) bool { return !t.IsZero() })).
		Return(expectedErr).Once()

	err := h.CreateUseCase().CleanupDeletedTeams(ctx, h.DefaultOlderThan())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CleanupUseCase")
	assert.Contains(t, err.Error(), expectedErr.Error())
	h.TeamRepo.AssertExpectations(t)
}
