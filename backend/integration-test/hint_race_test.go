package integration_test

import (
	"context"
	"sync"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHintUseCase_Unlock_Concurrent_DoubleSpending(t *testing.T) {
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	db, redisClient := redismock.NewClientMock()
	redisClient.ExpectDel("hint:lock:12345678-1234-5678-1234-567812345678").SetVal(0)
	uc := usecase.NewHintUseCase(f.HintRepo, f.HintUnlockRepo, f.AwardRepo, f.TxRepo, f.SolveRepo, db)

	_, team := f.CreateUserWithTeam(t, "hint_racer")
	award := &entity.Award{
		TeamId:      team.Id,
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
	hint := f.CreateHint(t, challenge.Id, 100, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)
	hintCh := make(chan *entity.Hint, 2)

	action := func() {
		defer wg.Done()
		h, err := uc.UnlockHint(ctx, team.Id, hint.Id)
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

	assert.Equal(t, 1, successes, "Only one unlock should succeed due to sufficient funds for only one")
	assert.Equal(t, 1, len(errors), "One unlock should fail with insufficient funds or already unlocked")

	unlocks, err := f.HintUnlockRepo.GetUnlockedHintIDs(ctx, team.Id, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, len(unlocks))

	checkTx, _ := f.TxRepo.BeginTx(ctx)
	finalScore, _ := f.TxRepo.GetTeamScoreTx(ctx, checkTx, team.Id)
	_ = checkTx.Rollback(ctx)

	assert.Equal(t, 0, finalScore, "Final score should be 0, not negative")
}
