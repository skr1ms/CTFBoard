package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create Tests

func TestSolveRepo_Create(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "solve")
	challenge := f.CreateChallenge(t, "solve_ch", 100)

	solve := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}

	err := f.SolveRepo.Create(ctx, solve)
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, solve.TeamId, solve.ChallengeId)
	require.NoError(t, err)
	assert.NotEmpty(t, gotSolve.Id)
	assert.False(t, gotSolve.SolvedAt.IsZero())
}

func TestSolveRepo_Create_Duplicate(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "duplicate")
	challenge := f.CreateChallenge(t, "duplicate_ch", 100)

	f.CreateSolve(t, user.Id, team.Id, challenge.Id)

	solve2 := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}
	err := f.SolveRepo.Create(ctx, solve2)
	assert.Error(t, err)
}

// GetByID Tests

func TestSolveRepo_GetByID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_by_id")
	challenge := f.CreateChallenge(t, "get_by_id_ch", 100)
	solve := f.CreateSolve(t, user.Id, team.Id, challenge.Id)

	gotSolve, err := f.SolveRepo.GetByID(ctx, solve.Id)
	require.NoError(t, err)
	assert.Equal(t, solve.Id, gotSolve.Id)
	assert.Equal(t, solve.UserId, gotSolve.UserId)
	assert.Equal(t, solve.TeamId, gotSolve.TeamId)
	assert.Equal(t, solve.ChallengeId, gotSolve.ChallengeId)
}

func TestSolveRepo_GetByID_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	nonExistentID := uuid.New()
	_, err := f.SolveRepo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

// GetByTeamAndChallenge Tests

func TestSolveRepo_GetByTeamAndChallenge(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_by_team")
	challenge := f.CreateChallenge(t, "get_by_team_ch", 100)
	solve := f.CreateSolve(t, user.Id, team.Id, challenge.Id)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.Id, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, solve.Id, gotSolve.Id)
	assert.Equal(t, team.Id, gotSolve.TeamId)
	assert.Equal(t, challenge.Id, gotSolve.ChallengeId)
}

func TestSolveRepo_GetByTeamAndChallenge_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "not_found")
	challenge := f.CreateChallenge(t, "not_found_ch", 100)

	_, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.Id, challenge.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

// GetByUserId Tests

func TestSolveRepo_GetByUserId(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_by_user")
	ch1 := f.CreateChallenge(t, "ch1", 100)
	ch2 := f.CreateChallenge(t, "ch2", 200)

	f.CreateSolve(t, user.Id, team.Id, ch1.Id)
	time.Sleep(1 * time.Second)
	f.CreateSolve(t, user.Id, team.Id, ch2.Id)

	solves, err := f.SolveRepo.GetByUserId(ctx, user.Id)
	require.NoError(t, err)
	assert.Len(t, solves, 2)
	assert.Equal(t, ch2.Id, solves[0].ChallengeId)
	assert.Equal(t, ch1.Id, solves[1].ChallengeId)
}

func TestSolveRepo_GetByUserId_Empty(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "empty")

	solves, err := f.SolveRepo.GetByUserId(ctx, user.Id)
	require.NoError(t, err)
	assert.Len(t, solves, 0)
}

// GetScoreboard Tests

func TestSolveRepo_GetScoreboard(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "score_1")
	u2, t2 := f.CreateUserWithTeam(t, "score_2")

	ch1 := f.CreateChallenge(t, "score_ch1", 100)
	ch2 := f.CreateChallenge(t, "score_ch2", 200)

	f.CreateSolve(t, u1.Id, t1.Id, ch1.Id)
	time.Sleep(10 * time.Millisecond)
	f.CreateSolve(t, u1.Id, t1.Id, ch2.Id)

	time.Sleep(10 * time.Millisecond)
	f.CreateSolve(t, u2.Id, t2.Id, ch1.Id)

	scoreboard, err := f.SolveRepo.GetScoreboard(ctx)
	require.NoError(t, err)
	assert.Len(t, scoreboard, 2)

	t1Found, t2Found := false, false
	for _, entry := range scoreboard {
		if entry.TeamId == t1.Id {
			assert.Equal(t, t1.Name, entry.TeamName)
			assert.Equal(t, 300, entry.Points)
			t1Found = true
		}
		if entry.TeamId == t2.Id {
			assert.Equal(t, t2.Name, entry.TeamName)
			assert.Equal(t, 100, entry.Points)
			t2Found = true
		}
	}
	assert.True(t, t1Found)
	assert.True(t, t2Found)
}

func TestSolveRepo_GetScoreboard_Empty(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "empty_score")

	scoreboard, err := f.SolveRepo.GetScoreboard(ctx)
	require.NoError(t, err)
	assert.Len(t, scoreboard, 1)
	assert.Equal(t, team.Name, scoreboard[0].TeamName)
	assert.Equal(t, 0, scoreboard[0].Points)
}

// GetFirstBlood Tests

func TestSolveRepo_GetFirstBlood(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "fb_1")
	u2, t2 := f.CreateUserWithTeam(t, "fb_2")
	ch := f.CreateChallenge(t, "fb_ch", 100)

	f.CreateSolve(t, u1.Id, t1.Id, ch.Id)
	time.Sleep(1 * time.Second)
	f.CreateSolve(t, u2.Id, t2.Id, ch.Id)

	firstBlood, err := f.SolveRepo.GetFirstBlood(ctx, ch.Id)
	require.NoError(t, err)
	assert.Equal(t, u1.Id, firstBlood.UserId)
	assert.Equal(t, u1.Username, firstBlood.Username)
	assert.Equal(t, t1.Id, firstBlood.TeamId)
	assert.Equal(t, t1.Name, firstBlood.TeamName)
}

func TestSolveRepo_GetFirstBlood_NoSolves(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch := f.CreateChallenge(t, "no_solves_ch", 100)

	_, err := f.SolveRepo.GetFirstBlood(ctx, ch.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

// GetScoreboardFrozen Tests

func TestSolveRepo_GetScoreboardFrozen(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "frozen")
	ch1 := f.CreateChallenge(t, "frozen_1", 100)
	ch2 := f.CreateChallenge(t, "frozen_2", 200)

	solve1 := f.CreateSolve(t, u1.Id, t1.Id, ch1.Id)

	backdated := time.Now().Add(-1 * time.Hour)
	_, err := f.Pool.Exec(ctx, "UPDATE solves SET solved_at = $1 WHERE id = $2", backdated, solve1.Id)
	require.NoError(t, err)

	freezeTime := time.Now().Add(-30 * time.Minute)

	f.CreateSolve(t, u1.Id, t1.Id, ch2.Id)

	scoreboard, err := f.SolveRepo.GetScoreboardFrozen(ctx, freezeTime)
	require.NoError(t, err)

	found := false
	for _, entry := range scoreboard {
		if entry.TeamId == t1.Id {
			assert.Equal(t, 100, entry.Points)
			found = true
		}
	}
	assert.True(t, found)
}

// CreateTx Tests

func TestSolveRepo_CreateTx(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "tx_create")
	ch := f.CreateChallenge(t, "tx_create_ch", 100)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	solve := &entity.Solve{
		UserId:      u.Id,
		TeamId:      tTeam.Id,
		ChallengeId: ch.Id,
	}

	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, tTeam.Id, ch.Id)
	require.NoError(t, err)
	assert.NotEmpty(t, gotSolve.Id)
	assert.Equal(t, u.Id, gotSolve.UserId)
}

func TestSolveRepo_CreateTx_Rollback(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "tx_rollback")
	ch := f.CreateChallenge(t, "tx_rollback_ch", 100)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	solve := &entity.Solve{
		UserId:      u.Id,
		TeamId:      tTeam.Id,
		ChallengeId: ch.Id,
	}

	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	require.NoError(t, err)

	err = tx.Rollback(ctx)
	require.NoError(t, err)

	_, err = f.SolveRepo.GetByTeamAndChallenge(ctx, tTeam.Id, ch.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

func TestSolveRepo_GetByTeamAndChallengeTx(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "get_tx")
	ch := f.CreateChallenge(t, "get_tx_ch", 100)
	f.CreateSolve(t, u.Id, tTeam.Id, ch.Id)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	gotSolve, err := f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, tTeam.Id, ch.Id)
	require.NoError(t, err)
	assert.Equal(t, tTeam.Id, gotSolve.TeamId)
	assert.Equal(t, ch.Id, gotSolve.ChallengeId)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestSolveRepo_GetByTeamAndChallengeTx_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, tTeam := f.CreateUserWithTeam(t, "not_found_tx")
	ch := f.CreateChallenge(t, "not_found_tx_ch", 100)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, tTeam.Id, ch.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

// GetTeamScoreTx Tests

func TestSolveRepo_GetTeamScoreTx(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "score_tx")
	ch1 := f.CreateChallenge(t, "score_tx_1", 100)
	ch2 := f.CreateChallenge(t, "score_tx_2", 200)

	f.CreateSolve(t, u.Id, tTeam.Id, ch1.Id)
	f.CreateSolve(t, u.Id, tTeam.Id, ch2.Id)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	score, err := f.TxRepo.GetTeamScoreTx(ctx, tx, tTeam.Id)
	require.NoError(t, err)
	assert.Equal(t, 300, score)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

// AtomicSubmitFlow Tests

func TestSolveRepo_AtomicSubmitFlow(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "atomic")
	initialValue, minValue, decay := 500, 100, 1
	ch := f.CreateDynamicChallenge(t, "atomic_ch", initialValue, minValue, decay)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	_, err = f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, tTeam.Id, ch.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))

	gotChallenge, err := f.TxRepo.GetChallengeByIDTx(ctx, tx, ch.Id)
	require.NoError(t, err)

	solve := &entity.Solve{
		UserId:      u.Id,
		TeamId:      tTeam.Id,
		ChallengeId: ch.Id,
	}
	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	require.NoError(t, err)

	solveCount := gotChallenge.SolveCount + 1
	newPoints := int(float64(gotChallenge.MinValue) + (float64(gotChallenge.InitialValue-gotChallenge.MinValue) / (1 + float64(solveCount-1)/float64(gotChallenge.Decay))))
	if newPoints < gotChallenge.MinValue {
		newPoints = gotChallenge.MinValue
	}

	_, err = f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, ch.Id)
	require.NoError(t, err)

	err = f.TxRepo.UpdateChallengePointsTx(ctx, tx, ch.Id, newPoints)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	finalChallenge, err := f.ChallengeRepo.GetByID(ctx, ch.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, finalChallenge.SolveCount)
	assert.Equal(t, newPoints, finalChallenge.Points)
	assert.Equal(t, initialValue, finalChallenge.Points)

	tx2, _ := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	finalSolve, err := f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx2, tTeam.Id, ch.Id)
	require.NoError(t, err)
	_ = tx2.Rollback(ctx)
	assert.Equal(t, u.Id, finalSolve.UserId)
}
