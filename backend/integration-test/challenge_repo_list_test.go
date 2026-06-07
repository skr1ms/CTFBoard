package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestChallengeRepo_GetAll_NoTeam(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "public_1", 100)
	ch2 := f.CreateChallenge(t, "public_2", 200)

	hiddenChallenge := &domain.Challenge{
		Title:       "HIDden Challenge",
		Description: "Description",
		Category:    "Pwn",
		Points:      300,
		FlagHash:    "hash3",
		State:       domain.ChallengeStateHidden,
	}
	hiddenChallenge.ID = uuid.New()
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.Create(txCtx, hiddenChallenge)
	})
	require.NoError(t, err)

	challenges, err := f.ChallengeRepo.GetAll(ctx, nil, nil)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, c := range challenges {
		ids[c.Challenge.ID] = true
	}

	assert.True(t, ids[ch1.ID], "ch1 should be in result")
	assert.True(t, ids[ch2.ID], "ch2 should be in result")

	for _, ch := range challenges {
		assert.Equal(t, domain.ChallengeStateVisible, ch.Challenge.State)
		assert.False(t, ch.Solved)
	}
}

func TestChallengeRepo_GetAll_WithTeam(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "team_user")

	err := f.UserRepo.UpdateTeamID(ctx, user.ID, &team.ID)
	require.NoError(t, err)

	ch1 := f.CreateChallenge(t, "ch_1", 100)
	ch2 := f.CreateChallenge(t, "ch_2", 200)

	f.CreateSolve(t, user.ID, team.ID, ch1.ID)

	challenges, err := f.ChallengeRepo.GetAll(ctx, &team.ID, nil)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, c := range challenges {
		ids[c.Challenge.ID] = true
	}

	assert.True(t, ids[ch1.ID], "ch1 should be in result")
	assert.True(t, ids[ch2.ID], "ch2 should be in result")

	solved := false

	for _, ch := range challenges {
		if ch.Challenge.ID == ch1.ID {
			assert.True(t, ch.Solved)

			solved = true
		} else {
			assert.False(t, ch.Solved)
		}
	}

	assert.True(t, solved)
}
