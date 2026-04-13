package loginlockout

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/testutil"
)

// redisClient is shared across all tests in this package via TestMain.
var testCtx = context.Background()

func setupTracker(t *testing.T) *Tracker {
	t.Helper()

	client, cleanup, err := testutil.StartRedisClient(testCtx)
	require.NoError(t, err, "start redis container")

	t.Cleanup(func() {
		cleanup()
	})

	return NewTracker(client, 5, 15*time.Minute)
}

func TestNewTracker_Defaults(t *testing.T) {
	t.Parallel()

	client, cleanup, err := testutil.StartRedisClient(testCtx)
	require.NoError(t, err)

	t.Cleanup(cleanup)

	// passing zeros → defaults
	tr := NewTracker(client, 0, 0)

	assert.Equal(t, defaultMax, tr.max)
	assert.Equal(t, defaultTTL, tr.ttl)
}

func TestIsLocked_NoAttempts(t *testing.T) {
	t.Parallel()

	tr := setupTracker(t)

	locked, err := tr.IsLocked(testCtx, "user@example.com")

	require.NoError(t, err)
	assert.False(t, locked)
}

func TestIsLocked_BelowMax(t *testing.T) {
	t.Parallel()

	tr := setupTracker(t)
	email := "below@example.com"

	for range 4 {
		require.NoError(t, tr.RecordFailed(testCtx, email))
	}

	locked, err := tr.IsLocked(testCtx, email)

	require.NoError(t, err)
	assert.False(t, locked)
}

func TestIsLocked_AtMax(t *testing.T) {
	t.Parallel()

	tr := setupTracker(t)
	email := "locked@example.com"

	for range 5 {
		require.NoError(t, tr.RecordFailed(testCtx, email))
	}

	locked, err := tr.IsLocked(testCtx, email)

	require.NoError(t, err)
	assert.True(t, locked)
}

func TestIsLocked_OverMax(t *testing.T) {
	t.Parallel()

	tr := setupTracker(t)
	email := "over@example.com"

	for range 10 {
		require.NoError(t, tr.RecordFailed(testCtx, email))
	}

	locked, err := tr.IsLocked(testCtx, email)

	require.NoError(t, err)
	assert.True(t, locked)
}

func TestRecordFailed_SetsTTL(t *testing.T) {
	t.Parallel()

	client, cleanup, err := testutil.StartRedisClient(testCtx)
	require.NoError(t, err)

	t.Cleanup(cleanup)

	tr := NewTracker(client, 5, 30*time.Second)
	email := "ttl@example.com"

	require.NoError(t, tr.RecordFailed(testCtx, email))

	key := "failed_login:" + email
	ttl := client.TTL(testCtx, key).Val()
	assert.Positive(t, ttl, "TTL should be set after first failed attempt")
	assert.LessOrEqual(t, ttl, 30*time.Second)
}

func TestClearFailed_ResetsLock(t *testing.T) {
	t.Parallel()

	tr := setupTracker(t)
	email := "clearme@example.com"

	for range 5 {
		require.NoError(t, tr.RecordFailed(testCtx, email))
	}

	locked, err := tr.IsLocked(testCtx, email)
	require.NoError(t, err)
	require.True(t, locked, "should be locked before clear")

	require.NoError(t, tr.ClearFailed(testCtx, email))

	locked, err = tr.IsLocked(testCtx, email)
	require.NoError(t, err)
	assert.False(t, locked, "should not be locked after clear")
}

func TestIsLocked_EmptyEmail_ReturnsFalse(t *testing.T) {
	t.Parallel()

	tr := setupTracker(t)

	locked, err := tr.IsLocked(testCtx, "")

	require.NoError(t, err)
	assert.False(t, locked)
}

func TestRecordFailed_EmptyEmail_ReturnsNil(t *testing.T) {
	t.Parallel()

	tr := setupTracker(t)

	err := tr.RecordFailed(testCtx, "")

	require.NoError(t, err)
}

func TestClearFailed_EmptyEmail_ReturnsNil(t *testing.T) {
	t.Parallel()

	tr := setupTracker(t)

	err := tr.ClearFailed(testCtx, "")

	require.NoError(t, err)
}
