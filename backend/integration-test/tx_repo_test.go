package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BeginTx Tests

func TestTxRepo_BeginTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	assert.NotNil(t, tx)

	err = tx.Rollback()
	require.NoError(t, err)
}

func TestTxRepo_BeginTx_Error(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)

	// Close DB to force error
	err := testDB.DB.Close()
	require.NoError(t, err)

	ctx := context.Background()
	tx, err := f.TxRepo.BeginTx(ctx)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

// RunTransaction Tests

func TestTxRepo_RunTransaction_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	user := f.CreateUser(t, "tx_run_user")
	executed := false

	err := f.TxRepo.RunTransaction(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		err := f.TxRepo.UpdateUserTeamIDTx(txCtx, tx, user.Id, nil)
		executed = true
		return err
	})

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestTxRepo_RunTransaction_Error(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	user := f.CreateUser(t, "tx_run_err_user")

	err := f.TxRepo.RunTransaction(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		err := f.TxRepo.UpdateUserTeamIDTx(txCtx, tx, user.Id, nil)
		require.NoError(t, err)
		return errors.New("forced error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forced error")
}

// CreateUserTx Tests

func TestTxRepo_CreateUserTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	user := &entity.User{
		Username:     "tx_user",
		Email:        "tx_user@example.com",
		PasswordHash: "hash",
		Role:         "user",
	}

	err = f.TxRepo.CreateUserTx(ctx, tx, user)
	require.NoError(t, err)
	assert.NotEmpty(t, user.Id)

	err = tx.Commit()
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.Username, gotUser.Username)
}

func TestTxRepo_CreateUserTx_Error_DuplicateEmail(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	existingUser := f.CreateUser(t, "existing")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

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
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "update_team")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = f.TxRepo.UpdateUserTeamIDTx(ctx, tx, user.Id, &team.Id)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)
}

func TestTxRepo_UpdateUserTeamIDTx_Error_InvalidUserID(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = f.TxRepo.UpdateUserTeamIDTx(ctx, tx, "invalid-uuid", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Parse UserID")
}

// CreateTeamTx Tests

func TestTxRepo_CreateTeamTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	user := f.CreateUser(t, "team_captain")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	team := &entity.Team{
		Name:        "TxTeam",
		InviteToken: "tx_token",
		CaptainId:   user.Id,
	}

	err = f.TxRepo.CreateTeamTx(ctx, tx, team)
	require.NoError(t, err)
	assert.NotEmpty(t, team.Id)

	err = tx.Commit()
	require.NoError(t, err)

	gotTeam, err := f.TeamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	assert.Equal(t, team.Name, gotTeam.Name)
}

func TestTxRepo_CreateTeamTx_Error_InvalidCaptainID(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	team := &entity.Team{
		Name:        "ErrorTeam",
		InviteToken: "error_token",
		CaptainId:   "invalid-uuid",
	}

	err = f.TxRepo.CreateTeamTx(ctx, tx, team)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Parse CaptainID")
}

// GetChallengeByIDTx Tests

func TestTxRepo_GetChallengeByIDTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "TxChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	gotChallenge, err := f.TxRepo.GetChallengeByIDTx(ctx, tx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, challenge.Id, gotChallenge.Id)
	assert.Equal(t, challenge.Title, gotChallenge.Title)
	assert.Equal(t, challenge.Points, gotChallenge.Points)

	err = tx.Commit()
	require.NoError(t, err)
}

func TestTxRepo_GetChallengeByIDTx_Error_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	_, err = f.TxRepo.GetChallengeByIDTx(ctx, tx, "00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
}

// IncrementChallengeSolveCountTx Tests

func TestTxRepo_IncrementChallengeSolveCountTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "IncrementChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	newCount, err := f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, newCount)

	newCount, err = f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, 2, newCount)

	err = tx.Commit()
	require.NoError(t, err)
}

func TestTxRepo_IncrementChallengeSolveCountTx_Error_InvalidID(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	_, err = f.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, "invalid-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ParseID")
}

// UpdateChallengePointsTx Tests

func TestTxRepo_UpdateChallengePointsTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "UpdatePoints", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = f.TxRepo.UpdateChallengePointsTx(ctx, tx, challenge.Id, 200)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	updated, err := f.ChallengeRepo.GetByID(ctx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, 200, updated.Points)
}

func TestTxRepo_UpdateChallengePointsTx_Error_InvalidID(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = f.TxRepo.UpdateChallengePointsTx(ctx, tx, "invalid-uuid", 200)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ParseID")
}

// CreateSolveTx Tests

func TestTxRepo_CreateSolveTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "solve_tx_user")
	challenge := f.CreateChallenge(t, "SolveTxChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	solve := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}

	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.Id, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, user.Id, gotSolve.UserId)
}

func TestTxRepo_CreateSolveTx_Error_InvalidUserID(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "solve_err_user")
	challenge := f.CreateChallenge(t, "SolveErrChallenge", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	solve := &entity.Solve{
		UserId:      "invalid-uuid",
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}

	err = f.TxRepo.CreateSolveTx(ctx, tx, solve)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Parse UserID")
}

// GetSolveByTeamAndChallengeTx Tests

func TestTxRepo_GetSolveByTeamAndChallengeTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "get_solve_tx")
	challenge := f.CreateChallenge(t, "GetSolveTx", 100)
	f.CreateSolve(t, user.Id, team.Id, challenge.Id)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	solve, err := f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, team.Id, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, team.Id, solve.TeamId)
	assert.Equal(t, challenge.Id, solve.ChallengeId)

	err = tx.Commit()
	require.NoError(t, err)
}

func TestTxRepo_GetSolveByTeamAndChallengeTx_Error_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "no_solve_tx")
	challenge := f.CreateChallenge(t, "NoSolveTx", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	_, err = f.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, team.Id, challenge.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

// GetTeamScoreTx Tests

func TestTxRepo_GetTeamScoreTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "score_tx_user")
	challenge1 := f.CreateChallenge(t, "Score1", 100)
	challenge2 := f.CreateChallenge(t, "Score2", 200)

	f.CreateSolve(t, user.Id, team.Id, challenge1.Id)
	f.CreateSolve(t, user.Id, team.Id, challenge2.Id)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	score, err := f.TxRepo.GetTeamScoreTx(ctx, tx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 300, score)

	err = tx.Commit()
	require.NoError(t, err)
}

func TestTxRepo_GetTeamScoreTx_Error_InvalidTeamID(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	_, err = f.TxRepo.GetTeamScoreTx(ctx, tx, "invalid-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Parse TeamID")
}

// CreateHintUnlockTx Tests

func TestTxRepo_CreateHintUnlockTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "hint_unlock_tx")
	challenge := f.CreateChallenge(t, "HintUnlockTx", 100)
	hint := f.CreateHint(t, challenge.Id, 10, 1)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = f.TxRepo.CreateHintUnlockTx(ctx, tx, team.Id, hint.Id)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	unlock, err := f.HintUnlockRepo.GetByTeamAndHint(ctx, team.Id, hint.Id)
	require.NoError(t, err)
	assert.Equal(t, team.Id, unlock.TeamId)
	assert.Equal(t, hint.Id, unlock.HintId)
}

func TestTxRepo_CreateHintUnlockTx_Error_InvalidTeamID(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "HintErrTx", 100)
	hint := f.CreateHint(t, challenge.Id, 10, 1)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = f.TxRepo.CreateHintUnlockTx(ctx, tx, "invalid-uuid", hint.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Parse TeamID")
}

// GetHintUnlockByTeamAndHintTx Tests

func TestTxRepo_GetHintUnlockByTeamAndHintTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "get_hint_unlock_tx")
	challenge := f.CreateChallenge(t, "GetHintUnlockTx", 100)
	hint := f.CreateHint(t, challenge.Id, 10, 1)

	tx1, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	err = f.TxRepo.CreateHintUnlockTx(ctx, tx1, team.Id, hint.Id)
	require.NoError(t, err)
	err = tx1.Commit()
	require.NoError(t, err)

	tx2, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()

	unlock, err := f.TxRepo.GetHintUnlockByTeamAndHintTx(ctx, tx2, team.Id, hint.Id)
	require.NoError(t, err)
	assert.Equal(t, team.Id, unlock.TeamId)
	assert.Equal(t, hint.Id, unlock.HintId)

	err = tx2.Commit()
	require.NoError(t, err)
}

func TestTxRepo_GetHintUnlockByTeamAndHintTx_Error_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "no_hint_unlock_tx")
	challenge := f.CreateChallenge(t, "NoHintUnlockTx", 100)
	hint := f.CreateHint(t, challenge.Id, 10, 1)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	_, err = f.TxRepo.GetHintUnlockByTeamAndHintTx(ctx, tx, team.Id, hint.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrHintNotFound))
}

// CreateAwardTx Tests

func TestTxRepo_CreateAwardTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "award_tx_user")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	award := &entity.Award{
		TeamId:      team.Id,
		Value:       50,
		Description: "Tx Award",
	}

	err = f.TxRepo.CreateAwardTx(ctx, tx, award)
	require.NoError(t, err)
	assert.NotEmpty(t, award.Id)

	err = tx.Commit()
	require.NoError(t, err)

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 50, total)
}

func TestTxRepo_CreateAwardTx_Error_InvalidTeamID(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	award := &entity.Award{
		TeamId:      "invalid-uuid",
		Value:       50,
		Description: "Error Award",
	}

	err = f.TxRepo.CreateAwardTx(ctx, tx, award)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Parse TeamID")
}

// LockTeamTx Tests

func TestTxRepo_LockTeamTx_Success(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "lock_team_tx")

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = f.TxRepo.LockTeamTx(ctx, tx, team.Id)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)
}

func TestTxRepo_LockTeamTx_Error_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = f.TxRepo.LockTeamTx(ctx, tx, "00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTeamNotFound))
}
