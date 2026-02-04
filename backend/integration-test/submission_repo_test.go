package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmissionRepo_Create_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "sub")
	challenge := f.CreateChallenge(t, "subch", 100)
	sub := &entity.Submission{
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "suberr")
	challenge := f.CreateChallenge(t, "suberrch", 100)
	sub := &entity.Submission{
		UserID:        uuid.New(),
		TeamID:        &team.ID,
		ChallengeID:   challenge.ID,
		SubmittedFlag: "x",
		IsCorrect:     false,
	}
	err := f.SubmissionRepo.Create(ctx, sub)
	assert.Error(t, err)
}

func TestSubmissionRepo_GetByChallenge_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "gbch")
	challenge := f.CreateChallenge(t, "gbch", 100)
	sub := &entity.Submission{UserID: user.ID, TeamID: &team.ID, ChallengeID: challenge.ID, SubmittedFlag: "x", IsCorrect: false}
	err := f.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err)
	list, err := f.SubmissionRepo.GetByChallenge(ctx, challenge.ID, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestSubmissionRepo_GetByChallenge_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	challenge := f.CreateChallenge(t, "gbcherr", 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.GetByChallenge(ctx, challenge.ID, 10, 0)
	assert.Error(t, err)
}

func TestSubmissionRepo_GetByUser_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "gbu")
	challenge := f.CreateChallenge(t, "gbu", 100)
	sub := &entity.Submission{UserID: user.ID, TeamID: &team.ID, ChallengeID: challenge.ID, SubmittedFlag: "x", IsCorrect: false}
	err := f.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err)
	list, err := f.SubmissionRepo.GetByUser(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestSubmissionRepo_GetByUser_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "gbuerr")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.GetByUser(ctx, user.ID, 10, 0)
	assert.Error(t, err)
}

func TestSubmissionRepo_GetByTeam_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "gbt")
	challenge := f.CreateChallenge(t, "gbt", 100)
	sub := &entity.Submission{UserID: user.ID, TeamID: &team.ID, ChallengeID: challenge.ID, SubmittedFlag: "x", IsCorrect: false}
	err := f.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err)
	list, err := f.SubmissionRepo.GetByTeam(ctx, team.ID, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestSubmissionRepo_GetByTeam_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	_, team := f.CreateUserWithTeam(t, "gbterr")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.GetByTeam(ctx, team.ID, 10, 0)
	assert.Error(t, err)
}

func TestSubmissionRepo_GetAll_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	list, err := f.SubmissionRepo.GetAll(ctx, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, list)
}

func TestSubmissionRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.GetAll(ctx, 10, 0)
	assert.Error(t, err)
}

func TestSubmissionRepo_CountByChallenge_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "cntch")
	challenge := f.CreateChallenge(t, "cntch", 100)
	sub := &entity.Submission{UserID: user.ID, TeamID: &team.ID, ChallengeID: challenge.ID, SubmittedFlag: "x", IsCorrect: false}
	err := f.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err)
	n, err := f.SubmissionRepo.CountByChallenge(ctx, challenge.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, int64(1))
}

func TestSubmissionRepo_CountByChallenge_Success_Empty(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	n, err := f.SubmissionRepo.CountByChallenge(ctx, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestSubmissionRepo_CountByChallenge_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	challenge := f.CreateChallenge(t, "cnterr", 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.CountByChallenge(ctx, challenge.ID)
	assert.Error(t, err)
}

func TestSubmissionRepo_GetStats_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "stats", 100)
	stats, err := f.SubmissionRepo.GetStats(ctx, challenge.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.Total, 0)
}

func TestSubmissionRepo_GetStats_Success_NoSubmissions(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "statsempty", 100)
	stats, err := f.SubmissionRepo.GetStats(ctx, challenge.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Correct)
	assert.Equal(t, 0, stats.Incorrect)
}

func TestSubmissionRepo_GetStats_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	challenge := f.CreateChallenge(t, "statserr", 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SubmissionRepo.GetStats(ctx, challenge.ID)
	assert.Error(t, err)
}
