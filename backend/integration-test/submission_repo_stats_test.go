package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestSubmissionRepo_CountByChallenge_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "cntch")
	challenge := f.CreateChallenge(t, "cntch", 100)
	sub := &domain.Submission{UserID: user.ID, TeamID: &team.ID, ChallengeID: challenge.ID, SubmittedFlag: "x", IsCorrect: false}
	err := f.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err)
	n, err := f.SubmissionRepo.CountByChallenge(ctx, challenge.ID, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, int64(1))
}

func TestSubmissionRepo_CountByChallenge_Success_Empty(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	n, err := f.SubmissionRepo.CountByChallenge(ctx, uuid.New(), nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestSubmissionRepo_GetStats_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "stats", 100)
	stats, err := f.SubmissionRepo.GetStats(ctx, challenge.ID, nil)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.Total, 0)
}

func TestSubmissionRepo_GetStats_Success_NoSubmissions(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "statsempty", 100)
	stats, err := f.SubmissionRepo.GetStats(ctx, challenge.ID, nil)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Correct)
	assert.Equal(t, 0, stats.Incorrect)
}

func TestSubmissionRepo_CountByTeamAndChallengeInWindow(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "wincount")
	ch := f.CreateChallenge(t, "wincount_ch", 100)

	for range 3 {
		sub := &domain.Submission{
			UserID:        user.ID,
			TeamID:        &team.ID,
			ChallengeID:   ch.ID,
			SubmittedFlag: "WRONG{flag}",
			IsCorrect:     false,
		}
		require.NoError(t, f.SubmissionRepo.Create(ctx, sub))
	}

	windowStart := timeNowMinus(5 * 60) // 5 minutes ago
	count, err := f.SubmissionRepo.CountSubmissionsByTeamAndChallengeInWindow(ctx, team.ID, ch.ID, windowStart)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(3))
}

func TestSubmissionRepo_CountFailedByIP(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "ipcount")
	ch := f.CreateChallenge(t, "ipcount_ch", 100)
	ip := "192.0.2.1"

	for range 4 {
		sub := &domain.Submission{
			UserID:        user.ID,
			TeamID:        &team.ID,
			ChallengeID:   ch.ID,
			SubmittedFlag: "WRONG{flag}",
			IsCorrect:     false,
			IP:            ip,
		}
		require.NoError(t, f.SubmissionRepo.Create(ctx, sub))
	}

	since := timeNowMinus(5 * 60)
	count, err := f.SubmissionRepo.CountFailedByIP(ctx, ip, since)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(4))
}
