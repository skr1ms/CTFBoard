package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
)

func TestTrackingRepo_CreateChallengeOpen_DeduplicatesByUserAndChallenge(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	repo := persistent.NewTrackingRepo(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "open_dedupe")
	secondUser := f.CreateUser(t, "open_dedupe_second")
	f.AddUserToTeam(t, secondUser.ID, team.ID)
	challenge := f.CreateChallenge(t, "open_dedupe", 100)
	secondChallenge := f.CreateChallenge(t, "open_dedupe_second", 100)

	open := &domain.ChallengeOpen{
		UserID:      user.ID,
		TeamID:      &team.ID,
		ChallengeID: challenge.ID,
		IP:          "192.0.2.1",
	}
	require.NoError(t, repo.CreateChallengeOpen(ctx, open))

	duplicateOpen := &domain.ChallengeOpen{
		UserID:      user.ID,
		TeamID:      &team.ID,
		ChallengeID: challenge.ID,
		IP:          "192.0.2.2",
	}
	require.NoError(t, repo.CreateChallengeOpen(ctx, duplicateOpen))

	count, err := repo.CountChallengeOpensByChallenge(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	secondUserOpen := &domain.ChallengeOpen{
		UserID:      secondUser.ID,
		TeamID:      &team.ID,
		ChallengeID: challenge.ID,
		IP:          "192.0.2.3",
	}
	require.NoError(t, repo.CreateChallengeOpen(ctx, secondUserOpen))

	count, err = repo.CountChallengeOpensByChallenge(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	otherChallengeOpen := &domain.ChallengeOpen{
		UserID:      user.ID,
		TeamID:      &team.ID,
		ChallengeID: secondChallenge.ID,
		IP:          "192.0.2.4",
	}
	require.NoError(t, repo.CreateChallengeOpen(ctx, otherChallengeOpen))

	count, err = repo.CountChallengeOpensByChallenge(ctx, secondChallenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
