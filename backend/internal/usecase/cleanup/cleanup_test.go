package cleanup

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	teamMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mock"
)

type cleanupTestDeps struct {
	teamRepo *teamMock.MockTeamRepository
}

type fakeTrackingRepo struct {
	deleteOlderThanRows        int64
	deleteOlderThanErr         error
	deleteChallengeOpensRows   int64
	deleteChallengeOpensErr    error
	deleteOlderThanCalled      bool
	deleteChallengeOpensCalled bool
	deleteOlderThanCutoff      time.Time
	deleteChallengeOpensCutoff time.Time
	createCalled               bool
	createChallengeOpenCalled  bool
	getByUserCalled            bool
	countByUserCalled          bool
	getChallengeOpensCalled    bool
	countChallengeOpensCalled  bool
}

func (f *fakeTrackingRepo) Create(_ context.Context, _ *domain.TrackingEntry) error {
	f.createCalled = true

	return nil
}

func (f *fakeTrackingRepo) GetByUser(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.TrackingEntry, error) {
	f.getByUserCalled = true

	return nil, nil
}

func (f *fakeTrackingRepo) CountByUser(_ context.Context, _ uuid.UUID) (int, error) {
	f.countByUserCalled = true

	return 0, nil
}

func (f *fakeTrackingRepo) DeleteOlderThan(_ context.Context, cutoffDate time.Time) (int64, error) {
	f.deleteOlderThanCalled = true
	f.deleteOlderThanCutoff = cutoffDate

	return f.deleteOlderThanRows, f.deleteOlderThanErr
}

func (f *fakeTrackingRepo) CreateChallengeOpen(_ context.Context, _ *domain.ChallengeOpen) error {
	f.createChallengeOpenCalled = true

	return nil
}

func (f *fakeTrackingRepo) GetChallengeOpensByChallenge(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.ChallengeOpen, error) {
	f.getChallengeOpensCalled = true

	return nil, nil
}

func (f *fakeTrackingRepo) DeleteChallengeOpensOlderThan(_ context.Context, cutoffDate time.Time) (int64, error) {
	f.deleteChallengeOpensCalled = true
	f.deleteChallengeOpensCutoff = cutoffDate

	return f.deleteChallengeOpensRows, f.deleteChallengeOpensErr
}

func (f *fakeTrackingRepo) CountChallengeOpensByChallenge(_ context.Context, _ uuid.UUID) (int, error) {
	f.countChallengeOpensCalled = true

	return 0, nil
}

func newCleanupTestDeps(t *testing.T) *cleanupTestDeps {
	t.Helper()

	return &cleanupTestDeps{teamRepo: teamMock.NewMockTeamRepository(t)}
}

func (d *cleanupTestDeps) createUseCase() *CleanupUseCase {
	return NewCleanupUseCase(CleanupDeps{TeamRepo: d.teamRepo})
}

func (d *cleanupTestDeps) createUseCaseWithTracking(trackingRepo *fakeTrackingRepo) *CleanupUseCase {
	return NewCleanupUseCase(CleanupDeps{TeamRepo: d.teamRepo, TrackingRepo: trackingRepo})
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

func TestCleanupUseCase_CleanupOldTracking_Success(t *testing.T) {
	t.Parallel()

	d := newCleanupTestDeps(t)
	ctx := context.Background()
	trackingRepo := &fakeTrackingRepo{
		deleteOlderThanRows:      11,
		deleteChallengeOpensRows: 7,
	}

	result, err := d.createUseCaseWithTracking(trackingRepo).CleanupOldTracking(ctx, defaultCleanupOlderThan())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(11), result.TrackingDeleted)
	assert.Equal(t, int64(7), result.ChallengeOpensDeleted)
	assert.True(t, trackingRepo.deleteOlderThanCalled)
	assert.True(t, trackingRepo.deleteChallengeOpensCalled)
	assert.False(t, trackingRepo.deleteOlderThanCutoff.IsZero())
	assert.False(t, trackingRepo.deleteChallengeOpensCutoff.IsZero())
}

func TestCleanupUseCase_CleanupOldTracking_NilRepo_Noops(t *testing.T) {
	t.Parallel()

	d := newCleanupTestDeps(t)

	result, err := d.createUseCase().CleanupOldTracking(context.Background(), defaultCleanupOlderThan())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Zero(t, result.TrackingDeleted)
	assert.Zero(t, result.ChallengeOpensDeleted)
}

func TestCleanupUseCase_CleanupOldTracking_DeleteTrackingError(t *testing.T) {
	t.Parallel()

	d := newCleanupTestDeps(t)
	expectedErr := errors.New("delete tracking failed")
	trackingRepo := &fakeTrackingRepo{deleteOlderThanErr: expectedErr}

	result, err := d.createUseCaseWithTracking(trackingRepo).CleanupOldTracking(context.Background(), defaultCleanupOlderThan())

	require.Error(t, err)
	require.NotNil(t, result)
	assert.Contains(t, err.Error(), expectedErr.Error())
	assert.True(t, trackingRepo.deleteOlderThanCalled)
	assert.False(t, trackingRepo.deleteChallengeOpensCalled)
	assert.Zero(t, result.TrackingDeleted)
	assert.Zero(t, result.ChallengeOpensDeleted)
}

func TestCleanupUseCase_CleanupOldTracking_DeleteChallengeOpensErrorKeepsTrackingCount(t *testing.T) {
	t.Parallel()

	d := newCleanupTestDeps(t)
	expectedErr := errors.New("delete opens failed")
	trackingRepo := &fakeTrackingRepo{
		deleteOlderThanRows:     9,
		deleteChallengeOpensErr: expectedErr,
	}

	result, err := d.createUseCaseWithTracking(trackingRepo).CleanupOldTracking(context.Background(), defaultCleanupOlderThan())

	require.Error(t, err)
	require.NotNil(t, result)
	assert.Contains(t, err.Error(), expectedErr.Error())
	assert.True(t, trackingRepo.deleteOlderThanCalled)
	assert.True(t, trackingRepo.deleteChallengeOpensCalled)
	assert.Equal(t, int64(9), result.TrackingDeleted)
	assert.Zero(t, result.ChallengeOpensDeleted)
}
