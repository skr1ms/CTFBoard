package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestSubmissionRepo_Create_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "sub")
	challenge := f.CreateChallenge(t, "subch", 100)
	sub := &domain.Submission{
		UserID:        user.ID,
		TeamID:        &team.ID,
		ChallengeID:   challenge.ID,
		SubmittedFlag: "flag{test}",
		IsCorrect:     false,
	}
	err := f.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err)
	assert.NotEmpty(t, sub.ID)
}

func TestSubmissionRepo_Create_Error_InvalidUserID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "suberr")
	challenge := f.CreateChallenge(t, "suberrch", 100)
	sub := &domain.Submission{
		UserID:        uuid.New(),
		TeamID:        &team.ID,
		ChallengeID:   challenge.ID,
		SubmittedFlag: "x",
		IsCorrect:     false,
	}
	err := f.SubmissionRepo.Create(ctx, sub)
	assert.Error(t, err)
}

func TestSubmissionRepo_Create_RatelimitedType(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "subrate")
	challenge := f.CreateChallenge(t, "subratech", 100)
	sub := &domain.Submission{
		UserID:        user.ID,
		TeamID:        &team.ID,
		ChallengeID:   challenge.ID,
		SubmittedFlag: "",
		IsCorrect:     false,
		Type:          domain.SubmissionTypeRatelimited,
	}
	err := f.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err)
	assert.NotEmpty(t, sub.ID)
	list, err := f.SubmissionRepo.GetByChallenge(ctx, challenge.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, domain.SubmissionTypeRatelimited, list[0].Type)
}

func TestSubmissionRepo_GetByChallenge_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "gbch")
	challenge := f.CreateChallenge(t, "gbch", 100)
	sub := &domain.Submission{UserID: user.ID, TeamID: &team.ID, ChallengeID: challenge.ID, SubmittedFlag: "x", IsCorrect: false}
	err := f.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err)
	list, err := f.SubmissionRepo.GetByChallenge(ctx, challenge.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestSubmissionRepo_GetByChallenge_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	challenge := f.CreateChallenge(t, "gbcherr", 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.GetByChallenge(ctx, challenge.ID, nil, 10, 0)
	assert.Error(t, err)
}

func TestSubmissionRepo_GetByUser_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "gbu")
	challenge := f.CreateChallenge(t, "gbu", 100)
	sub := &domain.Submission{UserID: user.ID, TeamID: &team.ID, ChallengeID: challenge.ID, SubmittedFlag: "x", IsCorrect: false}
	err := f.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err)
	list, err := f.SubmissionRepo.GetByUser(ctx, user.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestSubmissionRepo_GetByUser_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "gbuerr")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.GetByUser(ctx, user.ID, nil, 10, 0)
	assert.Error(t, err)
}

func TestSubmissionRepo_GetByTeam_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "gbt")
	challenge := f.CreateChallenge(t, "gbt", 100)
	sub := &domain.Submission{UserID: user.ID, TeamID: &team.ID, ChallengeID: challenge.ID, SubmittedFlag: "x", IsCorrect: false}
	err := f.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err)
	list, err := f.SubmissionRepo.GetByTeam(ctx, team.ID, nil, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestSubmissionRepo_GetByTeam_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	_, team := f.CreateUserWithTeam(t, "gbterr")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.GetByTeam(ctx, team.ID, nil, 10, 0)
	assert.Error(t, err)
}

func TestSubmissionRepo_GetAll_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	list, err := f.SubmissionRepo.GetAll(ctx, nil, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, list)
}

func TestSubmissionRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.GetAll(ctx, nil, 10, 0)
	assert.Error(t, err)
}

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

func TestSubmissionRepo_CountByChallenge_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	challenge := f.CreateChallenge(t, "cnterr", 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.CountByChallenge(ctx, challenge.ID, nil)
	assert.Error(t, err)
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

func TestSubmissionRepo_GetStats_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	challenge := f.CreateChallenge(t, "statserr", 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.GetStats(ctx, challenge.ID, nil)
	assert.Error(t, err)
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

func TestSubmissionRepo_AcquireAdvisoryLockForSubmit(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "advlock")
	ch := f.CreateChallenge(t, "advlock_ch", 100)

	// Advisory lock is transaction-scoped; verify it succeeds without error
	var lockErr error

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		lockErr = f.SubmissionRepo.AcquireAdvisoryLockForSubmit(txCtx, team.ID, ch.ID)

		return lockErr
	})
	require.NoError(t, err)
	require.NoError(t, lockErr)
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

func TestSubmissionRepo_SoftBanByUserID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "subban")
	ch := f.CreateChallenge(t, "subban_ch", 100)

	sub := &domain.Submission{
		UserID:        user.ID,
		TeamID:        &team.ID,
		ChallengeID:   ch.ID,
		SubmittedFlag: "WRONG{flag}",
		IsCorrect:     false,
	}
	require.NoError(t, f.SubmissionRepo.Create(ctx, sub))

	err := f.SubmissionRepo.SoftBanByUserID(ctx, user.ID)
	require.NoError(t, err)

	row := f.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND banned_user_id IS NOT NULL", user.ID)

	var count int
	require.NoError(t, row.Scan(&count))
	assert.GreaterOrEqual(t, count, 1)
}

func TestSubmissionRepo_RestoreByBannedUserID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "subrestore")
	ch := f.CreateChallenge(t, "subrestore_ch", 100)

	sub := &domain.Submission{
		UserID:        user.ID,
		TeamID:        &team.ID,
		ChallengeID:   ch.ID,
		SubmittedFlag: "WRONG{flag}",
		IsCorrect:     false,
	}
	require.NoError(t, f.SubmissionRepo.Create(ctx, sub))
	require.NoError(t, f.SubmissionRepo.SoftBanByUserID(ctx, user.ID))

	err := f.SubmissionRepo.RestoreByBannedUserID(ctx, user.ID)
	require.NoError(t, err)

	row := f.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND banned_user_id IS NULL", user.ID)

	var count int
	require.NoError(t, row.Scan(&count))
	assert.GreaterOrEqual(t, count, 1)
}

// timeNowMinus returns a time.Time that is `seconds` seconds before now.
func timeNowMinus(seconds int) time.Time {
	return time.Now().Add(-time.Duration(seconds) * time.Second)
}
