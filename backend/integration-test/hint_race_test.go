package integration_test

import (
	"context"
	"sync"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHintUseCase_Unlock_Concurrent_DoubleSpending(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	_, redisClient := redismock.NewClientMock()
	redisClient.ExpectDel("hint:lock:12345678-1234-5678-1234-567812345678").SetVal(0)
	uc := challenge.NewHintUseCase(challenge.HintDeps{
		HintRepo: f.HintRepo, AwardRepo: f.AwardRepo,
		TM: f.TM, SolveRepo: f.SolveRepo,
		CompRepo: f.CompetitionRepo, TeamRepo: f.TeamRepo,
		UserRepo:        f.UserRepo,
		ChallengeRepo:   f.ChallengeRepo,
		ScoreboardCache: nil,
	})

	user, team, challenge, hint := setupHintRaceTest(t, f, ctx)

	successes, errors := runConcurrentUnlocks(uc, ctx, user.ID, team.ID, challenge.ID, hint.ID)

	verifyHintUnlockResults(t, f, ctx, team, challenge, successes, errors)
}

func setupHintRaceTest(t *testing.T, f *TestFixture, ctx context.Context) (*entity.User, *entity.Team, *entity.Challenge, *entity.Hint) {
	t.Helper()
	user, team := f.CreateUserWithTeam(t, "hint_racer")
	award := &entity.Award{
		TeamID:      team.ID,
		Value:       100,
		Description: "Initial Funding",
	}
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.AwardRepo.Create(txCtx, award)
	})
	require.NoError(t, err)

	challenge := f.CreateChallenge(t, "HintRaceChall", 500)
	hint := f.CreateHint(t, challenge.ID, 100, 1)

	return user, team, challenge, hint
}

func runConcurrentUnlocks(uc *challenge.HintUseCase, ctx context.Context, userID, teamID, challengeID, hintID uuid.UUID) (int, []error) {
	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)
	hintCh := make(chan *entity.Hint, 2)

	action := func() {
		defer wg.Done()
		h, err := uc.UnlockHint(ctx, userID, teamID, challengeID, hintID)
		if err != nil {
			errCh <- err
		} else {
			hintCh <- h
		}
	}

	go action()
	go action()

	wg.Wait()
	close(errCh)
	close(hintCh)

	var successes int
	for range hintCh {
		successes++
	}

	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	return successes, errors
}

func verifyHintUnlockResults(t *testing.T, f *TestFixture, ctx context.Context, team *entity.Team, challenge *entity.Challenge, successes int, errors []error) {
	t.Helper()
	assert.Equal(t, 1, successes, "Only one unlock should succeed due to sufficient funds for only one")
	assert.Equal(t, 1, len(errors), "One unlock should fail with insufficient funds or already unlocked")

	unlocks, err := f.HintRepo.GetUnlockedHintIDs(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(unlocks))

	score, err := f.SolveRepo.GetTeamScore(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, score, "Final score should be 0, not negative")
}
