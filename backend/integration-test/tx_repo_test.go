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
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "tx_run_user")
	executed := false

	err := f.TxRepo.RunTransaction(ctx, func(txCtx context.Context, tx pgx.Tx) error {
		err := f.TxRepo.UpdateUserTeamIDTx(txCtx, tx, user.Id, nil)
		executed = true
		return err
	})

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestTxRepo_RunTransaction_Error(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "tx_run_err_user")

	err := f.TxRepo.RunTransaction(ctx, func(txCtx context.Context, tx pgx.Tx) error {
		err := f.TxRepo.UpdateUserTeamIDTx(txCtx, tx, user.Id, nil)
		require.NoError(t, err)
		return errors.New("forced error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forced error")
}

// CreateUserTx Tests

func TestTxRepo_CreateUserTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	user := &entity.User{
		Username:     "tx_user",
		Email:        "tx_user@example.com",
		PasswordHash: "hash",
		Role:         "user",
	}

	err = f.TxRepo.CreateUserTx(ctx, tx, user)
	require.NoError(t, err)
	assert.NotEmpty(t, user.Id)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.Username, gotUser.Username)
}

func TestTxRepo_CreateUserTx_Error_DuplicateEmail(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	existingUser := f.CreateUser(t, "existing")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	user := &entity.User{
		Username:     "different_username",
		Email:        existingUser.Email,
		PasswordHash: "hash",
		Role:         "user",
	}

	err = f.TxRepo.CreateUserTx(ctx, tx, user)
	assert.Error(t, err)
}

// UpdateUserTeamIDTx Tests

func TestTxRepo_UpdateUserTeamIDTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "update_team")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.UpdateUserTeamIDTx(ctx, tx, user.Id, &team.Id)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_UpdateUserTeamIDTx_Error_InvalidUserID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	nonExistentUserID := uuid.New()
	err = f.TxRepo.UpdateUserTeamIDTx(ctx, tx, nonExistentUserID, nil)
	assert.Error(t, err)
}

// CreateTeamTx Tests

func TestTxRepo_CreateTeamTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "team_captain")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	team := &entity.Team{
		Name:        "TxTeam",
		InviteToken: uuid.New(),
		CaptainId:   user.Id,
	}

	err = f.TxRepo.CreateTeamTx(ctx, tx, team)
	require.NoError(t, err)
	assert.NotEmpty(t, team.Id)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	gotTeam, err := f.TeamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	assert.Equal(t, team.Name, gotTeam.Name)
}

func TestTxRepo_CreateTeamTx_Error_InvalidCaptainID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	nonExistentCaptainID := uuid.New()
	team := &entity.Team{
		Name:        "ErrorTeam",
		InviteToken: uuid.New(),
		CaptainId:   nonExistentCaptainID,
	}

	err = f.TxRepo.CreateTeamTx(ctx, tx, team)
	assert.Error(t, err)
}

// GetChallengeByIDTx Tests

func TestTxRepo_GetChallengeByIDTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "TxChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	gotChallenge, err := f.TxRepo.GetChallengeByIDTx(ctx, tx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, challenge.Id, gotChallenge.Id)
	assert.Equal(t, challenge.Title, gotChallenge.Title)
	assert.Equal(t, challenge.Points, gotChallenge.Points)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetChallengeByIDTx_Error_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = f.TxRepo.GetChallengeByIDTx(ctx, tx, uuid.Nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
}

// IncrementChallengeSolveCountTx Tests

func TestTxRepo_IncrementChallengeSolveCountTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "IncrementChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	newCount, err := f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, newCount)

	newCount, err = f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, 2, newCount)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_IncrementChallengeSolveCountTx_Error_InvalidID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, uuid.New())
	assert.Error(t, err)
}

// UpdateChallengePointsTx Tests

func TestTxRepo_UpdateChallengePointsTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "UpdatePoints", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.UpdateChallengePointsTx(ctx, tx, challenge.Id, 200)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	updated, err := f.ChallengeRepo.GetByID(ctx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, 200, updated.Points)
}

func TestTxRepo_UpdateChallengePointsTx_Error_InvalidID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.UpdateChallengePointsTx(ctx, tx, uuid.New(), 200)
	assert.Error(t, err)
}

// CreateSolveTx Tests

func TestTxRepo_CreateSolveTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "solve_tx_user")
	challenge := f.CreateChallenge(t, "SolveTxChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	solve := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}

	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.Id, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, user.Id, gotSolve.UserId)
}

func TestTxRepo_CreateSolveTx_Error_InvalidUserID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "solve_err_user")
	challenge := f.CreateChallenge(t, "SolveErrChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	solve := &entity.Solve{
		UserId:      uuid.New(),
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}

	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	assert.Error(t, err)
}

// GetSolveByTeamAndChallengeTx Tests

func TestTxRepo_GetSolveByTeamAndChallengeTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_solve_tx")
	challenge := f.CreateChallenge(t, "GetSolveTx", 100)
	f.CreateSolve(t, user.Id, team.Id, challenge.Id)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	solve, err := f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, team.Id, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, team.Id, solve.TeamId)
	assert.Equal(t, challenge.Id, solve.ChallengeId)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetSolveByTeamAndChallengeTx_Error_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "no_solve_tx")
	challenge := f.CreateChallenge(t, "NoSolveTx", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, team.Id, challenge.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

// GetTeamScoreTx Tests

func TestTxRepo_GetTeamScoreTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "score_tx_user")
	challenge1 := f.CreateChallenge(t, "Score1", 100)
	challenge2 := f.CreateChallenge(t, "Score2", 200)

	f.CreateSolve(t, user.Id, team.Id, challenge1.Id)
	f.CreateSolve(t, user.Id, team.Id, challenge2.Id)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	score, err := f.TxRepo.GetTeamScoreTx(ctx, tx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 300, score)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetTeamScoreTx_NonExistentTeam(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	// Non-existent team should return 0 score
	score, err := f.TxRepo.GetTeamScoreTx(ctx, tx, uuid.New())
	assert.NoError(t, err)
	assert.Equal(t, 0, score)
}

// CreateHintUnlockTx Tests

func TestTxRepo_CreateHintUnlockTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "hint_unlock_tx")
	challenge := f.CreateChallenge(t, "HintUnlockTx", 100)
	hint := f.CreateHint(t, challenge.Id, 10, 1)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.CreateHintUnlockTx(ctx, tx, team.Id, hint.Id)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	unlock, err := f.HintUnlockRepo.GetByTeamAndHint(ctx, team.Id, hint.Id)
	require.NoError(t, err)
	assert.Equal(t, team.Id, unlock.TeamId)
	assert.Equal(t, hint.Id, unlock.HintId)
}

func TestTxRepo_CreateHintUnlockTx_Error_InvalidTeamID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "HintErrTx", 100)
	hint := f.CreateHint(t, challenge.Id, 10, 1)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.CreateHintUnlockTx(ctx, tx, uuid.New(), hint.Id)
	assert.Error(t, err)
}

// GetHintUnlockByTeamAndHintTx Tests

func TestTxRepo_GetHintUnlockByTeamAndHintTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_hint_unlock_tx")
	challenge := f.CreateChallenge(t, "GetHintUnlockTx", 100)
	hint := f.CreateHint(t, challenge.Id, 10, 1)

	tx1, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	err = f.TxRepo.CreateHintUnlockTx(ctx, tx1, team.Id, hint.Id)
	require.NoError(t, err)
	err = tx1.Commit(ctx)
	require.NoError(t, err)

	tx2, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback(ctx) }()

	unlock, err := f.TxRepo.GetHintUnlockByTeamAndHintTx(ctx, tx2, team.Id, hint.Id)
	require.NoError(t, err)
	assert.Equal(t, team.Id, unlock.TeamId)
	assert.Equal(t, hint.Id, unlock.HintId)

	err = tx2.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetHintUnlockByTeamAndHintTx_Error_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "no_hint_unlock_tx")
	challenge := f.CreateChallenge(t, "NoHintUnlockTx", 100)
	hint := f.CreateHint(t, challenge.Id, 10, 1)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = f.TxRepo.GetHintUnlockByTeamAndHintTx(ctx, tx, team.Id, hint.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrHintNotFound))
}

// CreateAwardTx Tests

func TestTxRepo_CreateAwardTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "award_tx_user")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	award := &entity.Award{
		TeamId:      team.Id,
		Value:       50,
		Description: "Tx Award",
	}

	err = f.TxRepo.CreateAwardTx(ctx, tx, award)
	require.NoError(t, err)
	assert.NotEmpty(t, award.Id)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 50, total)
}

func TestTxRepo_CreateAwardTx_Error_InvalidTeamID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	award := &entity.Award{
		TeamId:      uuid.New(),
		Value:       50,
		Description: "Error Award",
	}

	err = f.TxRepo.CreateAwardTx(ctx, tx, award)
	assert.Error(t, err)
}

// LockTeamTx Tests

func TestTxRepo_LockTeamTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "lock_team_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.LockTeamTx(ctx, tx, team.Id)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_LockTeamTx_Error_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.LockTeamTx(ctx, tx, uuid.Nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// BeginSerializableTx Tests

func TestTxRepo_BeginSerializableTx_Success(t *testing.T) {
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
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "lock_user_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.LockUserTx(ctx, tx, user.Id)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_LockUserTx_Error_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.LockUserTx(ctx, tx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserNotFound))
}

// GetTeamByNameTx Tests

func TestTxRepo_GetTeamByNameTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_team_name_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	gotTeam, err := f.TxRepo.GetTeamByNameTx(ctx, tx, team.Name)
	require.NoError(t, err)
	assert.Equal(t, team.Id, gotTeam.Id)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetTeamByNameTx_Error_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = f.TxRepo.GetTeamByNameTx(ctx, tx, "NonExistentTeam")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// GetTeamByInviteTokenTx Tests

func TestTxRepo_GetTeamByInviteTokenTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_team_token_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	gotTeam, err := f.TxRepo.GetTeamByInviteTokenTx(ctx, tx, team.InviteToken)
	require.NoError(t, err)
	assert.Equal(t, team.Id, gotTeam.Id)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetTeamByInviteTokenTx_Error_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = f.TxRepo.GetTeamByInviteTokenTx(ctx, tx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// GetUsersByTeamIDTx Tests

func TestTxRepo_GetUsersByTeamIDTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_users_team_tx")

	f.AddUserToTeam(t, user.Id, team.Id)

	user2 := f.CreateUser(t, "member2_tx")
	f.AddUserToTeam(t, user2.Id, team.Id)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	users, err := f.TxRepo.GetUsersByTeamIDTx(ctx, tx, team.Id)
	require.NoError(t, err)
	assert.Len(t, users, 2)

	ids := []uuid.UUID{users[0].Id, users[1].Id}
	assert.Contains(t, ids, user.Id)
	assert.Contains(t, ids, user2.Id)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_GetUsersByTeamIDTx_Error_Query(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tx, err := f.TxRepo.BeginTx(context.Background())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(context.Background()) }()

	_, err = f.TxRepo.GetUsersByTeamIDTx(ctx, tx, uuid.New())
	assert.Error(t, err)
}

// DeleteTeamTx Tests

func TestTxRepo_DeleteTeamTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "del_team_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.DeleteTeamTx(ctx, tx, team.Id)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	_, err = f.TeamRepo.GetByID(ctx, team.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

func TestTxRepo_DeleteTeamTx_Error_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.DeleteTeamTx(ctx, tx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// UpdateTeamCaptainTx Tests

func TestTxRepo_UpdateTeamCaptainTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	captain, team := f.CreateUserWithTeam(t, "cap_transfer_tx")
	newCap := f.CreateUser(t, "new_cap_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.UpdateTeamCaptainTx(ctx, tx, team.Id, newCap.Id)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	updatedTeam, err := f.TeamRepo.GetByID(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, newCap.Id, updatedTeam.CaptainId)
	assert.NotEqual(t, captain.Id, updatedTeam.CaptainId)
}

func TestTxRepo_UpdateTeamCaptainTx_Error_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.UpdateTeamCaptainTx(ctx, tx, uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// SoftDeleteTeamTx Tests

func TestTxRepo_SoftDeleteTeamTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "soft_del_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.SoftDeleteTeamTx(ctx, tx, team.Id)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_SoftDeleteTeamTx_Error_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = f.TxRepo.SoftDeleteTeamTx(ctx, tx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}

// CreateTeamAuditLogTx Tests

func TestTxRepo_CreateTeamAuditLogTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	captain, team := f.CreateUserWithTeam(t, "audit_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	log := &entity.TeamAuditLog{
		TeamId:  team.Id,
		UserId:  captain.Id,
		Action:  "TEST_ACTION",
		Details: map[string]interface{}{"foo": "bar"},
	}

	err = f.TxRepo.CreateTeamAuditLogTx(ctx, tx, log)
	require.NoError(t, err)
	assert.NotEmpty(t, log.Id)

	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func TestTxRepo_CreateTeamAuditLogTx_Error_InvalidReference(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	log := &entity.TeamAuditLog{
		TeamId: uuid.New(),
		UserId: uuid.New(),
		Action: "TEST",
	}

	err = f.TxRepo.CreateTeamAuditLogTx(ctx, tx, log)
	assert.Error(t, err)
}
