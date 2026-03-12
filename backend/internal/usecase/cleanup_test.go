package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mocks"
)

type cleanupTestDeps struct {
	teamRepo *mocks.MockTeamRepository
}

func newCleanupTestDeps(t *testing.T) *cleanupTestDeps {
	t.Helper()
	return &cleanupTestDeps{teamRepo: mocks.NewMockTeamRepository(t)}
}

func (d *cleanupTestDeps) createUseCase() *CleanupUseCase {
	return NewCleanupUseCase(CleanupDeps{TeamRepo: d.teamRepo})
}

func defaultCleanupOlderThan() time.Duration {
	return 24 * time.Hour
}

func TestCleanupUseCase_CleanupDeletedTeams_Success(t *testing.T) {
	t.Parallel()
	d := newCleanupTestDeps(t)
	ctx := context.Background()

	d.teamRepo.EXPECT().
		HardDeleteTeams(ctx, mock.MatchedBy(func(t interface{ IsZero() bool }) bool { return !t.IsZero() })).
		Return(nil).Once()

	err := d.createUseCase().CleanupDeletedTeams(ctx, defaultCleanupOlderThan())
	assert.NoError(t, err)
	d.teamRepo.AssertExpectations(t)
}

func TestCleanupUseCase_CleanupDeletedTeams_Error(t *testing.T) {
	t.Parallel()
	d := newCleanupTestDeps(t)
	ctx := context.Background()
	expectedErr := errors.New("db error")

	d.teamRepo.EXPECT().
		HardDeleteTeams(ctx, mock.MatchedBy(func(t interface{ IsZero() bool }) bool { return !t.IsZero() })).
		Return(expectedErr).Once()

	err := d.createUseCase().CleanupDeletedTeams(ctx, defaultCleanupOlderThan())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CleanupUseCase")
	assert.Contains(t, err.Error(), expectedErr.Error())
	d.teamRepo.AssertExpectations(t)
}
