package integration_test

import (
	"context"
	"sync"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/challenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHintUseCase_Unlock_Concurrent_DoubleSpending(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	db, redisClient := redismock.NewClientMock()
	redisClient.ExpectDel("hint:lock:12345678-1234-5678-1234-567812345678").SetVal(0)
	uc := challenge.NewHintUseCase(f.HintRepo, f.HintUnlockRepo, f.AwardRepo, f.TxRepo, f.SolveRepo, db)

	team, challenge, hint := setupHintRaceTest(t, f, ctx)

	successes, errors := runConcurrentUnlocks(uc, ctx, team.ID, hint.ID)

	verifyHintUnlockResults(t, f, ctx, team, challenge, successes, errors)
}

func setupHintRaceTest(t *testing.T, f *TestFixture, ctx context.Context) (*entity.Team, *entity.Challenge, *entity.Hint) {
	t.Helper()
	_, team := f.CreateUserWithTeam(t, "hint_racer")
	award := &entity.Award{
		TeamID:      team.ID,
		Value:       100,
		Description: "Initial Funding",
	}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	err = f.TxRepo.CreateAwardTx(ctx, tx, award)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	challenge := f.CreateChallenge(t, "HintRaceChall", 500)
	hint := f.CreateHint(t, challenge.ID, 100, 1)

	return team, challenge, hint
}

func runConcurrentUnlocks(uc *challenge.HintUseCase, ctx context.Context, teamID, hintID uuid.UUID) (int, []error) {
	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)
	hintCh := make(chan *entity.Hint, 2)

	action := func() {
		defer wg.Done()
		h, err := uc.UnlockHint(ctx, teamID, hintID)
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

	unlocks, err := f.HintUnlockRepo.GetUnlockedHintIDs(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(unlocks))

	checkTx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = checkTx.Rollback(ctx) }() //nolint:errcheck
	finalScore, err := f.TxRepo.GetTeamScoreTx(ctx, checkTx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, finalScore, "Final score should be 0, not negative")
}
