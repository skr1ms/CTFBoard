package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

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
