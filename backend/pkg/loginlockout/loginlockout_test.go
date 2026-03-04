package loginlockout

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEmail = "user@example.com"

func TestNewTracker_ZeroMaxAttempts_UsesDefault(t *testing.T) {
	t.Parallel()
	db, _ := redismock.NewClientMock()
	tracker := NewTracker(db, 0, 0)
	assert.Equal(t, defaultMax, tracker.max)
	assert.Equal(t, defaultTTL, tracker.ttl)
}

func TestNewTracker_CustomValues(t *testing.T) {
	t.Parallel()
	db, _ := redismock.NewClientMock()
	tracker := NewTracker(db, 10, 5*time.Minute)
	assert.Equal(t, 10, tracker.max)
	assert.Equal(t, 5*time.Minute, tracker.ttl)
}

func TestIsLocked_NoFailures_ReturnsFalse(t *testing.T) {
	t.Parallel()
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("failed_login:" + testEmail).SetErr(redis.Nil)

	tracker := NewTracker(db, 5, time.Minute)
	locked, err := tracker.IsLocked(context.Background(), testEmail)
	require.NoError(t, err)
	assert.False(t, locked)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIsLocked_BelowMax_ReturnsFalse(t *testing.T) {
	t.Parallel()
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("failed_login:" + testEmail).SetVal("3")

	tracker := NewTracker(db, 5, time.Minute)
	locked, err := tracker.IsLocked(context.Background(), testEmail)
	require.NoError(t, err)
	assert.False(t, locked)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIsLocked_AtMax_ReturnsTrue(t *testing.T) {
	t.Parallel()
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("failed_login:" + testEmail).SetVal("5")

	tracker := NewTracker(db, 5, time.Minute)
	locked, err := tracker.IsLocked(context.Background(), testEmail)
	require.NoError(t, err)
	assert.True(t, locked)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIsLocked_AboveMax_ReturnsTrue(t *testing.T) {
	t.Parallel()
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("failed_login:" + testEmail).SetVal("10")

	tracker := NewTracker(db, 5, time.Minute)
	locked, err := tracker.IsLocked(context.Background(), testEmail)
	require.NoError(t, err)
	assert.True(t, locked)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIsLocked_RedisError_ReturnsError(t *testing.T) {
	t.Parallel()
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("failed_login:" + testEmail).SetErr(errors.New("redis unavailable"))

	tracker := NewTracker(db, 5, time.Minute)
	locked, err := tracker.IsLocked(context.Background(), testEmail)
	require.Error(t, err)
	assert.False(t, locked)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordFailed_FirstFailure_IncrsAndSetsTTL(t *testing.T) {
	t.Parallel()
	db, mock := redismock.NewClientMock()
	key := "failed_login:" + testEmail

	// Pipeline: INCR returns 1, TTL returns -1 (key just created, no expiry)
	mock.ExpectIncr(key).SetVal(1)
	mock.ExpectTTL(key).SetVal(-1 * time.Second)
	mock.ExpectExpire(key, defaultTTL).SetVal(true)

	tracker := NewTracker(db, 5, 0)
	err := tracker.RecordFailed(context.Background(), testEmail)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordFailed_SubsequentFailure_NoExpireCall(t *testing.T) {
	t.Parallel()
	db, mock := redismock.NewClientMock()
	key := "failed_login:" + testEmail

	// Pipeline: INCR returns 2, TTL returns 600s (already has expiry)
	mock.ExpectIncr(key).SetVal(2)
	mock.ExpectTTL(key).SetVal(600 * time.Second)

	tracker := NewTracker(db, 5, defaultTTL)
	err := tracker.RecordFailed(context.Background(), testEmail)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordFailed_PipelineError_ReturnsError(t *testing.T) {
	t.Parallel()
	db, mock := redismock.NewClientMock()
	key := "failed_login:" + testEmail

	mock.ExpectIncr(key).SetErr(errors.New("redis down"))

	tracker := NewTracker(db, 5, defaultTTL)
	err := tracker.RecordFailed(context.Background(), testEmail)
	require.Error(t, err)
}
