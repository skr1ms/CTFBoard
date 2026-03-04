package integration_test

import (
	"context"
	"sync"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmissionRace_ConcurrentWrongFlagSubmits_NoDuplicateCountingOrPanic(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "sub_race")
	challenge := f.CreateChallenge(t, "sub_race_chall", 100)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			sub := &entity.Submission{
				UserID:        user.ID,
				TeamID:        &team.ID,
				ChallengeID:   challenge.ID,
				SubmittedFlag: "flag{wrong_answer}",
				IsCorrect:     false,
			}
			if err := f.SubmissionRepo.Create(ctx, sub); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected error in concurrent submission: %v", err)
	}

	count, err := f.SubmissionRepo.CountByUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(goroutines), count, "submission count must match exactly the number of goroutines")
}

func TestSubmissionRace_ConcurrentSubmissions_MultipleChallenges(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	const workers = 5
	users := make([]*entity.User, workers)
	teams := make([]*entity.Team, workers)
	for i := 0; i < workers; i++ {
		suffix := "mrace" + string(rune('a'+i))
		u, tm := f.CreateUserWithTeam(t, suffix)
		users[i], teams[i] = u, tm
	}
	challenge := f.CreateChallenge(t, "sub_multi_race", 50)

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			sub := &entity.Submission{
				UserID:        users[i].ID,
				TeamID:        &teams[i].ID,
				ChallengeID:   challenge.ID,
				SubmittedFlag: "flag{multi_race}",
				IsCorrect:     false,
			}
			_ = f.SubmissionRepo.Create(ctx, sub) //nolint:errcheck
		}()
	}
	wg.Wait()

	total, err := f.SubmissionRepo.CountByChallenge(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(workers), total)
}
