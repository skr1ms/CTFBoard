package integration_test

import (
	"context"
	"testing"

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
