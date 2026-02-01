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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "solve")
	challenge := f.CreateChallenge(t, "solve_ch", 100)

	solve := &entity.Solve{
		UserID:      user.ID,
		TeamID:      team.ID,
		ChallengeID: challenge.ID,
	}

	err := f.SolveRepo.Create(ctx, solve)
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, solve.TeamID, solve.ChallengeID)
	require.NoError(t, err)
	assert.NotEmpty(t, gotSolve.ID)
	assert.False(t, gotSolve.SolvedAt.IsZero())
}

func TestSolveRepo_Create_Duplicate(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "duplicate")
	challenge := f.CreateChallenge(t, "duplicate_ch", 100)

	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	solve2 := &entity.Solve{
		UserID:      user.ID,
		TeamID:      team.ID,
		ChallengeID: challenge.ID,
	}
	err := f.SolveRepo.Create(ctx, solve2)
	assert.Error(t, err)
}

// GetByID Tests

func TestSolveRepo_GetByID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_by_ID")
	challenge := f.CreateChallenge(t, "get_by_ID_ch", 100)
	solve := f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	gotSolve, err := f.SolveRepo.GetByID(ctx, solve.ID)
	require.NoError(t, err)
	assert.Equal(t, solve.ID, gotSolve.ID)
	assert.Equal(t, solve.UserID, gotSolve.UserID)
	assert.Equal(t, solve.TeamID, gotSolve.TeamID)
	assert.Equal(t, solve.ChallengeID, gotSolve.ChallengeID)
}

func TestSolveRepo_GetByID_NotFound(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_by_team")
	challenge := f.CreateChallenge(t, "get_by_team_ch", 100)
	solve := f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, solve.ID, gotSolve.ID)
	assert.Equal(t, team.ID, gotSolve.TeamID)
	assert.Equal(t, challenge.ID, gotSolve.ChallengeID)
}

func TestSolveRepo_GetByTeamAndChallenge_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "not_found")
	challenge := f.CreateChallenge(t, "not_found_ch", 100)

	_, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

// GetByUserID Tests

func TestSolveRepo_GetByUserID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_by_user")
	ch1 := f.CreateChallenge(t, "ch1", 100)
	ch2 := f.CreateChallenge(t, "ch2", 200)

	f.CreateSolve(t, user.ID, team.ID, ch1.ID)
	time.Sleep(1 * time.Second)
	f.CreateSolve(t, user.ID, team.ID, ch2.ID)

	solves, err := f.SolveRepo.GetByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, solves, 2)
	assert.Equal(t, ch2.ID, solves[0].ChallengeID)
	assert.Equal(t, ch1.ID, solves[1].ChallengeID)
}

func TestSolveRepo_GetByUserID_Empty(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "empty")

	solves, err := f.SolveRepo.GetByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, solves, 0)
}

// GetAll Tests

func TestSolveRepo_GetAll_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "get_all_1")
	u2, t2 := f.CreateUserWithTeam(t, "get_all_2")
	ch1 := f.CreateChallenge(t, "get_all_ch1", 100)
	ch2 := f.CreateChallenge(t, "get_all_ch2", 200)

	f.CreateSolve(t, u1.ID, t1.ID, ch1.ID)
	f.CreateSolve(t, u2.ID, t2.ID, ch2.ID)

	solves, err := f.SolveRepo.GetAll(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(solves), 2)
}

func TestSolveRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	solves, err := f.SolveRepo.GetAll(ctx)
	assert.Error(t, err)
	assert.Nil(t, solves)
}

// GetScoreboard Tests

func TestSolveRepo_GetScoreboard(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "score_1")
	u2, t2 := f.CreateUserWithTeam(t, "score_2")

	ch1 := f.CreateChallenge(t, "score_ch1", 100)
	ch2 := f.CreateChallenge(t, "score_ch2", 200)

	f.CreateSolve(t, u1.ID, t1.ID, ch1.ID)
	time.Sleep(10 * time.Millisecond)
	f.CreateSolve(t, u1.ID, t1.ID, ch2.ID)

	time.Sleep(10 * time.Millisecond)
	f.CreateSolve(t, u2.ID, t2.ID, ch1.ID)

	scoreboard, err := f.SolveRepo.GetScoreboard(ctx)
	require.NoError(t, err)
	assert.Len(t, scoreboard, 2)

	t1Found, t2Found := false, false
	for _, entry := range scoreboard {
		if entry.TeamID == t1.ID {
			assert.Equal(t, t1.Name, entry.TeamName)
			assert.Equal(t, 300, entry.Points)
			t1Found = true
		}
		if entry.TeamID == t2.ID {
			assert.Equal(t, t2.Name, entry.TeamName)
			assert.Equal(t, 100, entry.Points)
			t2Found = true
		}
	}
	assert.True(t, t1Found)
	assert.True(t, t2Found)
}

func TestSolveRepo_GetScoreboard_Empty(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "fb_1")
	u2, t2 := f.CreateUserWithTeam(t, "fb_2")
	ch := f.CreateChallenge(t, "fb_ch", 100)

	f.CreateSolve(t, u1.ID, t1.ID, ch.ID)
	time.Sleep(1 * time.Second)
	f.CreateSolve(t, u2.ID, t2.ID, ch.ID)

	firstBlood, err := f.SolveRepo.GetFirstBlood(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, u1.ID, firstBlood.UserID)
	assert.Equal(t, u1.Username, firstBlood.Username)
	assert.Equal(t, t1.ID, firstBlood.TeamID)
	assert.Equal(t, t1.Name, firstBlood.TeamName)
}

func TestSolveRepo_GetFirstBlood_NoSolves(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch := f.CreateChallenge(t, "no_solves_ch", 100)

	_, err := f.SolveRepo.GetFirstBlood(ctx, ch.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

// GetScoreboardFrozen Tests

func TestSolveRepo_GetScoreboardFrozen(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u1, t1 := f.CreateUserWithTeam(t, "frozen")
	ch1 := f.CreateChallenge(t, "frozen_1", 100)
	ch2 := f.CreateChallenge(t, "frozen_2", 200)

	solve1 := f.CreateSolve(t, u1.ID, t1.ID, ch1.ID)

	backdated := time.Now().Add(-1 * time.Hour)
	_, err := f.Pool.Exec(ctx, "UPDATE solves SET solved_at = $1 WHERE ID = $2", backdated, solve1.ID)
	require.NoError(t, err)

	freezeTime := time.Now().Add(-30 * time.Minute)

	f.CreateSolve(t, u1.ID, t1.ID, ch2.ID)

	scoreboard, err := f.SolveRepo.GetScoreboardFrozen(ctx, freezeTime)
	require.NoError(t, err)

	found := false
	for _, entry := range scoreboard {
		if entry.TeamID == t1.ID {
			assert.Equal(t, 100, entry.Points)
			found = true
		}
	}
	assert.True(t, found)
}

// CreateTx Tests

func TestSolveRepo_CreateTx(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "tx_create")
	ch := f.CreateChallenge(t, "tx_create_ch", 100)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	solve := &entity.Solve{
		UserID:      u.ID,
		TeamID:      tTeam.ID,
		ChallengeID: ch.ID,
	}

	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, tTeam.ID, ch.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, gotSolve.ID)
	assert.Equal(t, u.ID, gotSolve.UserID)
}

func TestSolveRepo_CreateTx_Rollback(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "tx_rollback")
	ch := f.CreateChallenge(t, "tx_rollback_ch", 100)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	solve := &entity.Solve{
		UserID:      u.ID,
		TeamID:      tTeam.ID,
		ChallengeID: ch.ID,
	}

	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	require.NoError(t, err)

	err = tx.Rollback(ctx)
	require.NoError(t, err)

	_, err = f.SolveRepo.GetByTeamAndChallenge(ctx, tTeam.ID, ch.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

func TestSolveRepo_GetByTeamAndChallengeTx(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "get_tx")
	ch := f.CreateChallenge(t, "get_tx_ch", 100)
	f.CreateSolve(t, u.ID, tTeam.ID, ch.ID)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback after commit is expected to fail

	gotSolve, err := f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, tTeam.ID, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, tTeam.ID, gotSolve.TeamID)
	assert.Equal(t, ch.ID, gotSolve.ChallengeID)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestSolveRepo_GetByTeamAndChallengeTx_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, tTeam := f.CreateUserWithTeam(t, "not_found_tx")
	ch := f.CreateChallenge(t, "not_found_tx_ch", 100)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback after commit is expected to fail

	_, err = f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, tTeam.ID, ch.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

// GetTeamScoreTx Tests

func TestSolveRepo_GetTeamScoreTx(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "score_tx")
	ch1 := f.CreateChallenge(t, "score_tx_1", 100)
	ch2 := f.CreateChallenge(t, "score_tx_2", 200)

	f.CreateSolve(t, u.ID, tTeam.ID, ch1.ID)
	f.CreateSolve(t, u.ID, tTeam.ID, ch2.ID)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback after commit is expected to fail

	score, err := f.TxRepo.GetTeamScoreTx(ctx, tx, tTeam.ID)
	require.NoError(t, err)
	assert.Equal(t, 300, score)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

// AtomicSubmitFlow Tests

func TestSolveRepo_AtomicSubmitFlow(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	u, tTeam := f.CreateUserWithTeam(t, "atomic")
	initialValue, minValue, decay := 500, 100, 1
	ch := f.CreateDynamicChallenge(t, "atomic_ch", initialValue, minValue, decay)

	tx, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	_, err = f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, tTeam.ID, ch.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))

	gotChallenge, err := f.TxRepo.GetChallengeByIDTx(ctx, tx, ch.ID)
	require.NoError(t, err)

	solve := &entity.Solve{
		UserID:      u.ID,
		TeamID:      tTeam.ID,
		ChallengeID: ch.ID,
	}
	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	require.NoError(t, err)

	solveCount := gotChallenge.SolveCount + 1
	newPoints := int(float64(gotChallenge.MinValue) + (float64(gotChallenge.InitialValue-gotChallenge.MinValue) / (1 + float64(solveCount-1)/float64(gotChallenge.Decay))))
	if newPoints < gotChallenge.MinValue {
		newPoints = gotChallenge.MinValue
	}

	_, err = f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, ch.ID)
	require.NoError(t, err)

	err = f.TxRepo.UpdateChallengePointsTx(ctx, tx, ch.ID, newPoints)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	finalChallenge, err := f.ChallengeRepo.GetByID(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, finalChallenge.SolveCount)
	assert.Equal(t, newPoints, finalChallenge.Points)
	assert.Equal(t, initialValue, finalChallenge.Points)

	tx2, err := f.Pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored
	finalSolve, err := f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx2, tTeam.ID, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, finalSolve.UserID)
}
