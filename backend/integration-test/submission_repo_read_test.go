package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

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

func TestSubmissionRepo_GetAll_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	list, err := f.SubmissionRepo.GetAll(ctx, nil, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, list)
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
