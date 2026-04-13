package integration_test

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestHintUnlock_ConcurrentBalanceRace(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "hint_race")
	ch := f.CreateChallenge(t, "race_ch", 500)
	f.CreateSolve(t, user.ID, team.ID, ch.ID)

	score, err := f.SolveRepo.GetTeamScore(ctx, team.ID)
	require.NoError(t, err)
	require.Equal(t, 500, score, "initial score should be 500")

	hint1 := f.CreateHint(t, ch.ID, 300, 0)
	hint2 := f.CreateHint(t, ch.ID, 300, 1)

	var wg sync.WaitGroup

	errors := make(chan error, 2)

	unlockHintWithCharge := func(hintID uuid.UUID, cost int) error {
		return f.TM.Run(ctx, func(txCtx context.Context) error {
			keyBytes := team.ID[8:16]
			key := int64(binary.BigEndian.Uint64(keyBytes)) & 0x7FFFFFFFFFFFFFFF

			db := f.TM.DB(txCtx)
			if _, err := db.Exec(txCtx, "SELECT pg_advisory_xact_lock($1::bigint)", key); err != nil {
				return err
			}

			score, err := f.SolveRepo.GetTeamScore(txCtx, team.ID)
			if err != nil {
				return err
			}

			if score < cost {
				return apperr.ErrInsufficientPoints
			}

			award := &domain.Award{
				TeamID:      team.ID,
				Value:       -cost,
				Description: "Hint unlock test",
			}
			if err := f.AwardRepo.Create(txCtx, award); err != nil {
				return err
			}

			return f.HintRepo.CreateUnlock(txCtx, team.ID, hintID)
		})
	}

	wg.Add(2)

	go func() {
		defer wg.Done()

		errors <- unlockHintWithCharge(hint1.ID, 300)
	}()

	go func() {
		defer wg.Done()

		errors <- unlockHintWithCharge(hint2.ID, 300)
	}()

	wg.Wait()
	close(errors)

	errCount := 0
	successCount := 0

	for err := range errors {
		if err != nil {
			errCount++
		} else {
			successCount++
		}
	}

	require.Equal(t, 1, successCount, "exactly one unlock should succeed")
	require.Equal(t, 1, errCount, "exactly one unlock should fail with insufficient points")

	finalScore, err := f.SolveRepo.GetTeamScore(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 200, finalScore, "final score should be 500 - 300 = 200, not negative")

	unlocks, err := f.HintRepo.GetUnlockedHintIDs(ctx, team.ID, ch.ID)
	require.NoError(t, err)
	assert.Len(t, unlocks, 1, "exactly one hint should be unlocked")
}

func TestHintUnlock_SequentialCorrect(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "hint_seq")
	ch := f.CreateChallenge(t, "seq_ch", 1000)
	f.CreateSolve(t, user.ID, team.ID, ch.ID)

	hint1 := f.CreateHint(t, ch.ID, 200, 0)
	hint2 := f.CreateHint(t, ch.ID, 300, 1)

	award1 := &domain.Award{
		TeamID:      team.ID,
		Value:       -200,
		Description: "Hint 1",
	}
	err := f.AwardRepo.Create(ctx, award1)
	require.NoError(t, err)

	err = f.HintRepo.CreateUnlock(ctx, team.ID, hint1.ID)
	require.NoError(t, err)

	score1, err := f.SolveRepo.GetTeamScore(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 800, score1, "score after first unlock: 1000 - 200 = 800")

	award2 := &domain.Award{
		TeamID:      team.ID,
		Value:       -300,
		Description: "Hint 2",
	}
	err = f.AwardRepo.Create(ctx, award2)
	require.NoError(t, err)

	err = f.HintRepo.CreateUnlock(ctx, team.ID, hint2.ID)
	require.NoError(t, err)

	score2, err := f.SolveRepo.GetTeamScore(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 500, score2, "score after second unlock: 800 - 300 = 500")
}
