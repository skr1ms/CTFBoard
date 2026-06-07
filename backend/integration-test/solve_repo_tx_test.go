package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestSolveRepo_GetTeamScoreTx(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "score_tx")
	ch1 := f.CreateChallenge(t, "score_tx_1", 100)
	ch2 := f.CreateChallenge(t, "score_tx_2", 200)

	f.CreateSolve(t, u.ID, tTeam.ID, ch1.ID)
	f.CreateSolve(t, u.ID, tTeam.ID, ch2.ID)

	var score int

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		var err error

		score, err = f.SolveRepo.GetTeamScore(txCtx, tTeam.ID)

		return err
	})
	require.NoError(t, err)
	assert.Equal(t, 300, score)
}

func TestSolveRepo_AtomicSubmitFlow(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "atomic")
	initialValue, minValue, decay := 500, 100, 1
	ch := f.CreateDynamicChallenge(t, "atomic_ch", initialValue, minValue, decay)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		_, err := f.SolveRepo.GetByTeamAndChallengeForUpdate(txCtx, tTeam.ID, ch.ID)
		if err == nil {
			return errors.New("expected not found")
		}

		if !errors.Is(err, apperr.ErrSolveNotFound) {
			return err
		}

		gotChallenge, err := f.ChallengeRepo.GetByIDForUpdate(txCtx, ch.ID)
		if err != nil {
			return err
		}

		solve := &domain.Solve{UserID: u.ID, TeamID: tTeam.ID, ChallengeID: ch.ID}
		if err := f.SolveRepo.Create(txCtx, solve); err != nil {
			return err
		}

		solveCount := gotChallenge.SolveCount + 1

		newPoints := max(int(float64(gotChallenge.MinValue)+(float64(gotChallenge.InitialValue-gotChallenge.MinValue)/(1+float64(solveCount-1)/float64(gotChallenge.Decay)))), gotChallenge.MinValue)

		_, err = f.ChallengeRepo.IncrementSolveCount(txCtx, ch.ID)
		if err != nil {
			return err
		}

		return f.ChallengeRepo.UpdatePoints(txCtx, ch.ID, newPoints)
	})
	require.NoError(t, err)

	finalChallenge, err := f.ChallengeRepo.GetByID(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, finalChallenge.SolveCount)
	assert.Equal(t, initialValue, finalChallenge.Points)

	finalSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, tTeam.ID, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, finalSolve.UserID)
}
