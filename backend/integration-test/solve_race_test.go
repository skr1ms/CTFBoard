package integration_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolveUseCase_Create_Concurrent_DuplicateSubmission(t *testing.T) {
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	db, redisClient := redismock.NewClientMock()
	redisClient.ExpectDel("solve:lock:12345678-1234-5678-1234-567812345678").SetVal(0)
	uc := usecase.NewSolveUseCase(f.SolveRepo, f.ChallengeRepo, f.CompetitionRepo, f.TxRepo, db, nil)

	captain, team := f.CreateUserWithTeam(t, "solve_racer")
	u2 := f.CreateUser(t, "solve_racer_2")
	f.AddUserToTeam(t, u2.Id, team.Id)

	challenge := f.CreateChallenge(t, "SolveRaceChall", 100)

	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)

	submit := func(uID uuid.UUID) {
		defer wg.Done()
		solve := &entity.Solve{
			UserId:      uID,
			TeamId:      team.Id,
			ChallengeId: challenge.Id,
		}
		err := uc.Create(ctx, solve)
		if err != nil {
			errCh <- err
		}
	}

	go submit(captain.Id)
	go submit(u2.Id)

	wg.Wait()
	close(errCh)

	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	assert.Equal(t, 1, len(errors), "Exactly one submission should fail")

	var count int
	_ = f.Pool.QueryRow(ctx, "SELECT count(*) FROM solves WHERE team_id = $1 AND challenge_id = $2", team.Id, challenge.Id).Scan(&count)
	assert.Equal(t, 1, count, "Should be exactly 1 solve record")
}

func TestSolveUseCase_Create_Concurrent_DynamicDecay(t *testing.T) {
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	db, redisClient := redismock.NewClientMock()
	redisClient.ExpectDel("solve:lock:12345678-1234-5678-1234-567812345678").SetVal(0)
	uc := usecase.NewSolveUseCase(f.SolveRepo, f.ChallengeRepo, f.CompetitionRepo, f.TxRepo, db, nil)

	challenge := f.CreateDynamicChallenge(t, "DecayRace", 1000, 100, 10)

	concurrency := 5
	var wg sync.WaitGroup
	wg.Add(concurrency)

	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			suffix := fmt.Sprintf("decay_%d", idx)
			u, tm := f.CreateUserWithTeam(t, suffix)

			solve := &entity.Solve{
				UserId:      u.Id,
				TeamId:      tm.Id,
				ChallengeId: challenge.Id,
			}
			if err := uc.Create(ctx, solve); err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		assert.NoError(t, err)
	}

	finalChall, err := f.ChallengeRepo.GetByID(ctx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, concurrency, finalChall.SolveCount, "Solve count should match number of successes")
	assert.NotEqual(t, 1000, finalChall.Points, "Points should have decayed")
}
