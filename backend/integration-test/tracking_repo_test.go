package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
)

func TestTrackingRepo_Create_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "tracking_create")
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)
	ctx := context.Background()

	entry := &domain.TrackingEntry{
		UserID:    user.ID,
		IP:        "192.168.1.1",
		UserAgent: "test-agent/1.0",
	}
	err := trackingRepo.Create(ctx, entry)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, entry.ID)
}

func TestTrackingRepo_Create_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	user := NewTestFixture(testPool.Pool).CreateUser(t, "tracking_create_err")
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	entry := &domain.TrackingEntry{UserID: user.ID, IP: "1.2.3.4"}
	err := trackingRepo.Create(ctx, entry)
	assert.Error(t, err)
}

func TestTrackingRepo_GetByUser_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "tracking_get")
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)
	ctx := context.Background()

	for range 3 {
		entry := &domain.TrackingEntry{UserID: user.ID, IP: "10.0.0.1", UserAgent: "agent"}
		require.NoError(t, trackingRepo.Create(ctx, entry))
	}

	entries, err := trackingRepo.GetByUser(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	for _, e := range entries {
		assert.Equal(t, user.ID, e.UserID)
	}
}

func TestTrackingRepo_GetByUser_Error(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := trackingRepo.GetByUser(ctx, uuid.New(), 10, 0)
	assert.Error(t, err)
}

func TestTrackingRepo_CountByUser_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "tracking_count")
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)
	ctx := context.Background()

	for range 5 {
		entry := &domain.TrackingEntry{UserID: user.ID, IP: "172.16.0.1"}
		require.NoError(t, trackingRepo.Create(ctx, entry))
	}

	count, err := trackingRepo.CountByUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestTrackingRepo_CreateChallengeOpen_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "chall_open_create")
	challenge := f.CreateChallenge(t, "chall_open", 100)
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)
	ctx := context.Background()

	open := &domain.ChallengeOpen{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		IP:          "10.10.10.10",
	}
	err := trackingRepo.CreateChallengeOpen(ctx, open)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, open.ID)
}

func TestTrackingRepo_GetChallengeOpensByChallenge_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "chall_open_get")
	challenge := f.CreateChallenge(t, "chall_open_get", 100)
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)
	ctx := context.Background()

	for range 2 {
		open := &domain.ChallengeOpen{UserID: user.ID, ChallengeID: challenge.ID, IP: "1.1.1.1"}
		require.NoError(t, trackingRepo.CreateChallengeOpen(ctx, open))
	}

	opens, err := trackingRepo.GetChallengeOpensByChallenge(ctx, challenge.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, opens, 2)

	for _, o := range opens {
		assert.Equal(t, challenge.ID, o.ChallengeID)
	}
}

func TestTrackingRepo_CountByUser_Error(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := trackingRepo.CountByUser(ctx, uuid.New())
	assert.Error(t, err)
}

func TestTrackingRepo_CountChallengeOpensByChallenge_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "chall_open_count")
	challenge := f.CreateChallenge(t, "chall_open_count", 100)
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)
	ctx := context.Background()

	for range 3 {
		open := &domain.ChallengeOpen{UserID: user.ID, ChallengeID: challenge.ID, IP: "2.2.2.2"}
		require.NoError(t, trackingRepo.CreateChallengeOpen(ctx, open))
	}

	count, err := trackingRepo.CountChallengeOpensByChallenge(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestTrackingRepo_CountChallengeOpensByChallenge_Error(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := trackingRepo.CountChallengeOpensByChallenge(ctx, uuid.New())
	assert.Error(t, err)
}

func TestTrackingRepo_CreateChallengeOpen_Error(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "chall_open_err")
	challenge := f.CreateChallenge(t, "chall_open_err", 100)
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	open := &domain.ChallengeOpen{UserID: user.ID, ChallengeID: challenge.ID, IP: "3.3.3.3"}
	err := trackingRepo.CreateChallengeOpen(ctx, open)
	assert.Error(t, err)
}

func TestTrackingRepo_GetChallengeOpensByChallenge_Error(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	trackingRepo := persistent.NewTrackingRepo(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := trackingRepo.GetChallengeOpensByChallenge(ctx, uuid.New(), 10, 0)
	assert.Error(t, err)
}
