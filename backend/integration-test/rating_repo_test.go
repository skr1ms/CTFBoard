package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestRatingRepo_Upsert_Create(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "rating_create")
	challenge := f.CreateChallenge(t, "rating_create", 100)
	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	rating := &domain.Rating{
		ChallengeID: challenge.ID,
		UserID:      user.ID,
		TeamID:      team.ID,
		Value:       4,
		Review:      "good challenge",
	}

	err := f.RatingRepo.Upsert(ctx, rating)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, rating.ID)
	assert.False(t, rating.CreatedAt.IsZero())
}

func TestRatingRepo_Upsert_UpdateExisting(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "rating_upsert")
	challenge := f.CreateChallenge(t, "rating_upsert", 100)
	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	rating := &domain.Rating{
		ChallengeID: challenge.ID,
		UserID:      user.ID,
		TeamID:      team.ID,
		Value:       3,
		Review:      "meh",
	}
	require.NoError(t, f.RatingRepo.Upsert(ctx, rating))
	firstID := rating.ID

	// Update - same team+challenge, different value and review.
	rating.Value = 5
	rating.Review = "actually great"
	require.NoError(t, f.RatingRepo.Upsert(ctx, rating))

	// ID stays the same after upsert.
	assert.Equal(t, firstID, rating.ID)

	got, err := f.RatingRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, got.Value)
	assert.Equal(t, "actually great", got.Review)
}

func TestRatingRepo_GetByChallengeID_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "rating_get_by_ch", 100)

	// Create two users/teams, each rates the same challenge.
	u1, t1 := f.CreateUserWithTeam(t, "rating_get_by_ch_1")
	u2, t2 := f.CreateUserWithTeam(t, "rating_get_by_ch_2")
	f.CreateSolve(t, u1.ID, t1.ID, challenge.ID)
	f.CreateSolve(t, u2.ID, t2.ID, challenge.ID)

	require.NoError(t, f.RatingRepo.Upsert(ctx, &domain.Rating{
		ChallengeID: challenge.ID, UserID: u1.ID, TeamID: t1.ID, Value: 4,
	}))
	require.NoError(t, f.RatingRepo.Upsert(ctx, &domain.Rating{
		ChallengeID: challenge.ID, UserID: u2.ID, TeamID: t2.ID, Value: 2,
	}))

	ratings, err := f.RatingRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Len(t, ratings, 2)
}

func TestRatingRepo_GetByChallengeID_Empty(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "rating_get_empty", 100)

	ratings, err := f.RatingRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Empty(t, ratings)
}

func TestRatingRepo_GetByTeamAndChallenge_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "rating_by_team")
	challenge := f.CreateChallenge(t, "rating_by_team", 100)
	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	require.NoError(t, f.RatingRepo.Upsert(ctx, &domain.Rating{
		ChallengeID: challenge.ID, UserID: user.ID, TeamID: team.ID, Value: 5, Review: "excellent",
	}))

	got, err := f.RatingRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, got.Value)
	assert.Equal(t, "excellent", got.Review)
	assert.Equal(t, team.ID, got.TeamID)
	assert.Equal(t, challenge.ID, got.ChallengeID)
}

func TestRatingRepo_GetByTeamAndChallenge_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.RatingRepo.GetByTeamAndChallenge(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, apperr.ErrRatingNotFound)
}

func TestRatingRepo_GetAll_ReturnsAll(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "rating_get_all", 100)
	u1, t1 := f.CreateUserWithTeam(t, "rating_get_all_1")
	u2, t2 := f.CreateUserWithTeam(t, "rating_get_all_2")
	f.CreateSolve(t, u1.ID, t1.ID, challenge.ID)
	f.CreateSolve(t, u2.ID, t2.ID, challenge.ID)

	require.NoError(t, f.RatingRepo.Upsert(ctx, &domain.Rating{
		ChallengeID: challenge.ID, UserID: u1.ID, TeamID: t1.ID, Value: 3,
	}))
	require.NoError(t, f.RatingRepo.Upsert(ctx, &domain.Rating{
		ChallengeID: challenge.ID, UserID: u2.ID, TeamID: t2.ID, Value: 4,
	}))

	all, err := f.RatingRepo.GetAll(ctx)
	require.NoError(t, err)
	// At least the two we just inserted are present (other parallel tests may add more).
	assert.GreaterOrEqual(t, len(all), 2)
}

func TestRatingRepo_Upsert_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.RatingRepo.Upsert(ctx, &domain.Rating{
		ChallengeID: uuid.New(),
		UserID:      uuid.New(),
		TeamID:      uuid.New(),
		Value:       3,
	})
	assert.Error(t, err)
}
