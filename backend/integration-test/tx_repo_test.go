package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BeginTx Tests

func TestTxRepo_BeginTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	assert.NotNil(t, tx)

	err = tx.Rollback(ctx)
	require.NoError(t, err)
}

func TestTxRepo_BeginTx_Error(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	// Close Pool to force error
	testPool.Pool.Close()

	ctx := context.Background()
	tx, err := f.TxRepo.BeginTx(ctx)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

// RunTransaction Tests

func TestTxRepo_RunTransaction_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "tx_run_user")
	executed := false

	err := f.TxRepo.RunTransaction(ctx, func(txCtx context.Context, tx pgx.Tx) error {
		err := f.TxRepo.UpdateUserTeamIDTx(txCtx, tx, user.ID, nil)
		executed = true
		return err
	})

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestTxRepo_RunTransaction_Error(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "tx_run_err_user")

	err := f.TxRepo.RunTransaction(ctx, func(txCtx context.Context, tx pgx.Tx) error {
		err := f.TxRepo.UpdateUserTeamIDTx(txCtx, tx, user.ID, nil)
		require.NoError(t, err)
		return errors.New("forced error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forced error")
}

// CreateUserTx Tests

func TestTxRepo_CreateUserTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	user := &entity.User{
		Username:     "tx_user",
		Email:        "tx_user@example.com",
		PasswordHash: "hash",
		Role:         entity.RoleUser,
	}

	err = f.TxRepo.CreateUserTx(ctx, tx, user)
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.Username, gotUser.Username)
}

func TestTxRepo_CreateUserTx_Error_DuplicateEmail(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	existingUser := f.CreateUser(t, "existing")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	user := &entity.User{
		Username:     "different_username",
		Email:        existingUser.Email,
		PasswordHash: "hash",
		Role:         entity.RoleUser,
	}

	err = f.TxRepo.CreateUserTx(ctx, tx, user)
	assert.Error(t, err)
}

// UpdateUserTeamIDTx Tests

func TestTxRepo_UpdateUserTeamIDTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "update_team")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.UpdateUserTeamIDTx(ctx, tx, user.ID, &team.ID)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_UpdateUserTeamIDTx_Error_InvalidUserID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	nonExistentUserID := uuid.New()
	err = f.TxRepo.UpdateUserTeamIDTx(ctx, tx, nonExistentUserID, nil)
	assert.Error(t, err)
}

// CreateTeamTx Tests

func TestTxRepo_CreateTeamTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "team_captain")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	team := &entity.Team{
		Name:        "TxTeam",
		InviteToken: uuid.New(),
		CaptainID:   user.ID,
	}

	err = f.TxRepo.CreateTeamTx(ctx, tx, team)
	require.NoError(t, err)
	assert.NotEmpty(t, team.ID)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	gotTeam, err := f.TeamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	assert.Equal(t, team.Name, gotTeam.Name)
}

func TestTxRepo_CreateTeamTx_Error_InvalidCaptainID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	nonExistentCaptainID := uuid.New()
	team := &entity.Team{
		Name:        "ErrorTeam",
		InviteToken: uuid.New(),
		CaptainID:   nonExistentCaptainID,
	}

	err = f.TxRepo.CreateTeamTx(ctx, tx, team)
	assert.Error(t, err)
}

// GetChallengeByIDTx Tests

func TestTxRepo_GetChallengeByIDTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "TxChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	gotChallenge, err := f.TxRepo.GetChallengeByIDTx(ctx, tx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, challenge.ID, gotChallenge.ID)
	assert.Equal(t, challenge.Title, gotChallenge.Title)
	assert.Equal(t, challenge.Points, gotChallenge.Points)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetChallengeByIDTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	_, err = f.TxRepo.GetChallengeByIDTx(ctx, tx, uuid.Nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
}

// IncrementChallengeSolveCountTx Tests

func TestTxRepo_IncrementChallengeSolveCountTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "IncrementChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	newCount, err := f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, newCount)

	newCount, err = f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, newCount)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_IncrementChallengeSolveCountTx_Error_InvalidID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	_, err = f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, uuid.New())
	assert.Error(t, err)
}

// UpdateChallengePointsTx Tests

func TestTxRepo_UpdateChallengePointsTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "UpdatePoints", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.UpdateChallengePointsTx(ctx, tx, challenge.ID, 200)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	updated, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 200, updated.Points)
}

func TestTxRepo_UpdateChallengePointsTx_Error_InvalidID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.UpdateChallengePointsTx(ctx, tx, uuid.New(), 200)
	assert.Error(t, err)
}

// CreateSolveTx Tests

func TestTxRepo_CreateSolveTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "solve_tx_user")
	challenge := f.CreateChallenge(t, "SolveTxChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	solve := &entity.Solve{
		UserID:      user.ID,
		TeamID:      team.ID,
		ChallengeID: challenge.ID,
	}

	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, gotSolve.UserID)
}

func TestTxRepo_CreateSolveTx_Error_InvalidUserID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "solve_err_user")
	challenge := f.CreateChallenge(t, "SolveErrChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	solve := &entity.Solve{
		UserID:      uuid.New(),
		TeamID:      team.ID,
		ChallengeID: challenge.ID,
	}

	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	assert.Error(t, err)
}

// GetSolveByTeamAndChallengeTx Tests

func TestTxRepo_GetSolveByTeamAndChallengeTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_solve_tx")
	challenge := f.CreateChallenge(t, "GetSolveTx", 100)
	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	solve, err := f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, team.ID, solve.TeamID)
	assert.Equal(t, challenge.ID, solve.ChallengeID)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetSolveByTeamAndChallengeTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "no_solve_tx")
	challenge := f.CreateChallenge(t, "NoSolveTx", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	_, err = f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, team.ID, challenge.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

// GetTeamScoreTx Tests

func TestTxRepo_GetTeamScoreTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "score_tx_user")
	challenge1 := f.CreateChallenge(t, "Score1", 100)
	challenge2 := f.CreateChallenge(t, "Score2", 200)

	f.CreateSolve(t, user.ID, team.ID, challenge1.ID)
	f.CreateSolve(t, user.ID, team.ID, challenge2.ID)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	score, err := f.TxRepo.GetTeamScoreTx(ctx, tx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 300, score)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetTeamScoreTx_NonExistentTeam(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	// Non-existent team should return 0 score
	score, err := f.TxRepo.GetTeamScoreTx(ctx, tx, uuid.New())
	assert.NoError(t, err)
	assert.Equal(t, 0, score)
}

// CreateHintUnlockTx Tests

func TestTxRepo_CreateHintUnlockTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "hint_unlock_tx")
	challenge := f.CreateChallenge(t, "HintUnlockTx", 100)
	hint := f.CreateHint(t, challenge.ID, 10, 1)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.CreateHintUnlockTx(ctx, tx, team.ID, hint.ID)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	unlock, err := f.HintUnlockRepo.GetByTeamAndHint(ctx, team.ID, hint.ID)
	require.NoError(t, err)
	assert.Equal(t, team.ID, unlock.TeamID)
	assert.Equal(t, hint.ID, unlock.HintID)
}

func TestTxRepo_CreateHintUnlockTx_Error_InvalidTeamID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "HintErrTx", 100)
	hint := f.CreateHint(t, challenge.ID, 10, 1)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.CreateHintUnlockTx(ctx, tx, uuid.New(), hint.ID)
	assert.Error(t, err)
}

// GetHintUnlockByTeamAndHintTx Tests

func TestTxRepo_GetHintUnlockByTeamAndHintTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_hint_unlock_tx")
	challenge := f.CreateChallenge(t, "GetHintUnlockTx", 100)
	hint := f.CreateHint(t, challenge.ID, 10, 1)

	tx1, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	err = f.TxRepo.CreateHintUnlockTx(ctx, tx1, team.ID, hint.ID)
	require.NoError(t, err)
	err = tx1.Commit(ctx)
	require.NoError(t, err)

	tx2, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	unlock, err := f.TxRepo.GetHintUnlockByTeamAndHintTx(ctx, tx2, team.ID, hint.ID)
	require.NoError(t, err)
	assert.Equal(t, team.ID, unlock.TeamID)
	assert.Equal(t, hint.ID, unlock.HintID)

	err = tx2.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetHintUnlockByTeamAndHintTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "no_hint_unlock_tx")
	challenge := f.CreateChallenge(t, "NoHintUnlockTx", 100)
	hint := f.CreateHint(t, challenge.ID, 10, 1)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	_, err = f.TxRepo.GetHintUnlockByTeamAndHintTx(ctx, tx, team.ID, hint.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrHintNotFound))
}

// CreateAwardTx Tests

func TestTxRepo_CreateAwardTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "award_tx_user")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	award := &entity.Award{
		TeamID:      team.ID,
		Value:       50,
		Description: "Tx Award",
	}

	err = f.TxRepo.CreateAwardTx(ctx, tx, award)
	require.NoError(t, err)
	assert.NotEmpty(t, award.ID)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, total)
}

func TestTxRepo_CreateAwardTx_Error_InvalidTeamID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	award := &entity.Award{
		TeamID:      uuid.New(),
		Value:       50,
		Description: "Error Award",
	}

	err = f.TxRepo.CreateAwardTx(ctx, tx, award)
	assert.Error(t, err)
}

// LockTeamTx Tests

func TestTxRepo_LockTeamTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "lock_team_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.LockTeamTx(ctx, tx, team.ID)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_LockTeamTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.LockTeamTx(ctx, tx, uuid.Nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// BeginSerializableTx Tests

func TestTxRepo_BeginSerializableTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginSerializableTx(ctx)
	require.NoError(t, err)
	assert.NotNil(t, tx)

	err = tx.Rollback(ctx)
	require.NoError(t, err)
}

func TestTxRepo_BeginSerializableTx_Error(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	testPool.Pool.Close()

	ctx := context.Background()
	tx, err := f.TxRepo.BeginSerializableTx(ctx)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

// LockUserTx Tests

func TestTxRepo_LockUserTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "lock_user_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.LockUserTx(ctx, tx, user.ID)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_LockUserTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.LockUserTx(ctx, tx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

// GetTeamByNameTx Tests

func TestTxRepo_GetTeamByNameTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_team_name_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	gotTeam, err := f.TxRepo.GetTeamByNameTx(ctx, tx, team.Name)
	require.NoError(t, err)
	assert.Equal(t, team.ID, gotTeam.ID)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetTeamByNameTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	_, err = f.TxRepo.GetTeamByNameTx(ctx, tx, "NonExistentTeam")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// GetTeamByInviteTokenTx Tests

func TestTxRepo_GetTeamByInviteTokenTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_team_token_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	gotTeam, err := f.TxRepo.GetTeamByInviteTokenTx(ctx, tx, team.InviteToken)
	require.NoError(t, err)
	assert.Equal(t, team.ID, gotTeam.ID)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetTeamByInviteTokenTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	_, err = f.TxRepo.GetTeamByInviteTokenTx(ctx, tx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// GetUsersByTeamIDTx Tests

func TestTxRepo_GetUsersByTeamIDTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_users_team_tx")

	f.AddUserToTeam(t, user.ID, team.ID)

	user2 := f.CreateUser(t, "member2_tx")
	f.AddUserToTeam(t, user2.ID, team.ID)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	users, err := f.TxRepo.GetUsersByTeamIDTx(ctx, tx, team.ID)
	require.NoError(t, err)
	assert.Len(t, users, 2)

	IDs := []uuid.UUID{users[0].ID, users[1].ID}
	assert.Contains(t, IDs, user.ID)
	assert.Contains(t, IDs, user2.ID)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetUsersByTeamIDTx_Error_Query(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tx, err := f.TxRepo.BeginTx(context.Background())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(context.Background()) }() //nolint:errcheck // rollback in defer, error ignored

	_, err = f.TxRepo.GetUsersByTeamIDTx(ctx, tx, uuid.New())
	assert.Error(t, err)
}

// DeleteTeamTx Tests

func TestTxRepo_DeleteTeamTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "del_team_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.DeleteTeamTx(ctx, tx, team.ID)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	_, err = f.TeamRepo.GetByID(ctx, team.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

func TestTxRepo_DeleteTeamTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.DeleteTeamTx(ctx, tx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// UpdateTeamCaptainTx Tests

func TestTxRepo_UpdateTeamCaptainTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	captain, team := f.CreateUserWithTeam(t, "cap_transfer_tx")
	newCap := f.CreateUser(t, "new_cap_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.UpdateTeamCaptainTx(ctx, tx, team.ID, newCap.ID)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	updatedTeam, err := f.TeamRepo.GetByID(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, newCap.ID, updatedTeam.CaptainID)
	assert.NotEqual(t, captain.ID, updatedTeam.CaptainID)
}

func TestTxRepo_UpdateTeamCaptainTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.UpdateTeamCaptainTx(ctx, tx, uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// SoftDeleteTeamTx Tests

func TestTxRepo_SoftDeleteTeamTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "soft_del_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.SoftDeleteTeamTx(ctx, tx, team.ID)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_SoftDeleteTeamTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.SoftDeleteTeamTx(ctx, tx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// CreateTeamAuditLogTx Tests

func TestTxRepo_CreateTeamAuditLogTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	captain, team := f.CreateUserWithTeam(t, "audit_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	log := &entity.TeamAuditLog{
		TeamID:  team.ID,
		UserID:  captain.ID,
		Action:  "TEST_ACTION",
		Details: map[string]any{"foo": "bar"},
	}

	err = f.TxRepo.CreateTeamAuditLogTx(ctx, tx, log)
	require.NoError(t, err)
	assert.NotEmpty(t, log.ID)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_CreateTeamAuditLogTx_Error_InvalidReference(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	log := &entity.TeamAuditLog{
		TeamID: uuid.New(),
		UserID: uuid.New(),
		Action: "TEST",
	}

	err = f.TxRepo.CreateTeamAuditLogTx(ctx, tx, log)
	assert.Error(t, err)
}

// DeleteSolvesByTeamIDTx Tests

func TestTxRepo_DeleteSolvesByTeamIDTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "del_solves_tx")
	challenge := f.CreateChallenge(t, "DelSolvesTx", 100)
	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	err = f.TxRepo.DeleteSolvesByTeamIDTx(ctx, tx, team.ID)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	_, err = f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

func TestTxRepo_DeleteSolvesByTeamIDTx_Error_TxClosed(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "del_solves_err")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))

	err = f.TxRepo.DeleteSolvesByTeamIDTx(ctx, tx, team.ID)
	assert.Error(t, err)
}

// GetTeamByIDTx Tests

func TestTxRepo_GetTeamByIDTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_team_ID_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	gotTeam, err := f.TxRepo.GetTeamByIDTx(ctx, tx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, team.ID, gotTeam.ID)
	assert.Equal(t, team.Name, gotTeam.Name)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetTeamByIDTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	_, err = f.TxRepo.GetTeamByIDTx(ctx, tx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// GetSoloTeamByUserIDTx Tests

func TestTxRepo_GetSoloTeamByUserIDTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	user := f.CreateUser(t, "solo_getter")

	team := &entity.Team{
		Name:        "SoloGetterTeam",
		CaptainID:   user.ID,
		IsSolo:      true,
		InviteToken: uuid.New(),
	}
	err = f.TxRepo.CreateTeamTx(ctx, tx, team)
	require.NoError(t, err)

	err = f.TxRepo.UpdateUserTeamIDTx(ctx, tx, user.ID, &team.ID)
	require.NoError(t, err)

	gotTeam, err := f.TxRepo.GetSoloTeamByUserIDTx(ctx, tx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, team.ID, gotTeam.ID)
	assert.True(t, gotTeam.IsSolo)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetSoloTeamByUserIDTx_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	user := f.CreateUser(t, "no_solo_team")

	_, err = f.TxRepo.GetSoloTeamByUserIDTx(ctx, tx, user.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// CreateAuditLogTx Tests

func TestTxRepo_CreateAuditLogTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "audit_logger")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	log := &entity.AuditLog{
		UserID:     &user.ID,
		Action:     "TEST_ACTION",
		EntityType: "user",
		EntityID:   user.ID.String(),
		IP:         "127.0.0.1",
		Details:    map[string]any{"foo": "bar"},
	}

	err = f.TxRepo.CreateAuditLogTx(ctx, tx, log)
	require.NoError(t, err)
	assert.NotEmpty(t, log.ID)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_CreateAuditLogTx_Error_InvalidUser(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback in defer, error ignored

	invalidID := uuid.New()
	log := &entity.AuditLog{
		UserID:     &invalidID,
		Action:     "TEST",
		EntityType: "test",
		EntityID:   "123",
	}
	require.NoError(t, tx.Rollback(ctx))

	err = f.TxRepo.CreateAuditLogTx(ctx, tx, log)
	assert.Error(t, err)
}
