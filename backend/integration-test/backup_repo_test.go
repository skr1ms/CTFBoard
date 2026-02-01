package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupRepo_EraseAllTablesTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "erase_user")
	challenge := f.CreateChallenge(t, "erase_chall", 100)

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.EraseAllTablesTx(ctx, tx)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	_, err = f.UserRepo.GetByID(ctx, user.ID)
	assert.Error(t, err)
	_, err = f.TeamRepo.GetByID(ctx, team.ID)
	assert.Error(t, err)
	_, err = f.ChallengeRepo.GetByID(ctx, challenge.ID)
	assert.Error(t, err)
}

func TestBackupRepo_EraseAllTablesTx_Error_ClosedTx(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))

	err = f.BackupRepo.EraseAllTablesTx(ctx, tx)
	assert.Error(t, err)
}

func TestBackupRepo_ImportCompetitionTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	comp := &entity.Competition{
		ID:   1,
		Name: "Updated CTF",
		Mode: "flexible",
	}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportCompetitionTx(ctx, tx, comp)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	got, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Updated CTF", got.Name)
}

func TestBackupRepo_ImportCompetitionTx_NilCompetition(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportCompetitionTx(ctx, tx, nil)
	require.NoError(t, err)
}

func TestBackupRepo_ImportChallengesTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challengeID := uuid.New()
	data := &entity.BackupData{
		Challenges: []entity.ChallengeExport{
			{
				Challenge: entity.Challenge{
					ID:           challengeID,
					Title:        "Backup Chall",
					Description:  "Desc",
					Category:     "Web",
					Points:       150,
					FlagHash:     "hash",
					InitialValue: 150,
					MinValue:     150,
					Decay:        0,
				},
				Hints: []entity.Hint{},
			},
		},
	}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportChallengesTx(ctx, tx, data)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	got, err := f.ChallengeRepo.GetByID(ctx, challengeID)
	require.NoError(t, err)
	assert.Equal(t, "Backup Chall", got.Title)
	assert.Equal(t, 150, got.Points)
}

func TestBackupRepo_ImportChallengesTx_Error_InvalidHintChallengeID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challengeID := uuid.New()
	nonexistentChallengeID := uuid.New()
	data := &entity.BackupData{
		Challenges: []entity.ChallengeExport{
			{
				Challenge: entity.Challenge{ID: challengeID, Title: "Ch", Description: "D", Category: "W", Points: 100, FlagHash: "h", InitialValue: 100, MinValue: 100, Decay: 0},
				Hints: []entity.Hint{
					{ID: uuid.New(), ChallengeID: nonexistentChallengeID, Content: "hint", Cost: 0, OrderIndex: 0},
				},
			},
		},
	}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportChallengesTx(ctx, tx, data)
	assert.Error(t, err)
}

func TestBackupRepo_ImportTeamsTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, _ := f.CreateUserWithTeam(t, "import_team_user")
	teamID := uuid.New()
	data := &entity.BackupData{
		Teams: []entity.TeamExport{
			{
				Team: entity.Team{
					ID:          teamID,
					Name:        "Imported Team",
					CaptainID:   user.ID,
					InviteToken: uuid.New(),
					IsSolo:      false,
					IsBanned:    false,
					IsHidden:    false,
					CreatedAt:   time.Now(),
				},
				MemberIDs: []uuid.UUID{user.ID},
			},
		},
	}
	opts := entity.ImportOptions{ConflictMode: entity.ConflictModeOverwrite}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportTeamsTx(ctx, tx, data, opts)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	got, err := f.TeamRepo.GetByID(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, "Imported Team", got.Name)
}

func TestBackupRepo_ImportTeamsTx_Error_InvalidCaptainID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	teamID := uuid.New()
	data := &entity.BackupData{
		Teams: []entity.TeamExport{
			{
				Team: entity.Team{
					ID:          teamID,
					Name:        "Bad Team",
					CaptainID:   uuid.New(),
					InviteToken: uuid.New(),
					CreatedAt:   time.Now(),
				},
			},
		},
	}
	opts := entity.ImportOptions{ConflictMode: entity.ConflictModeOverwrite}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportTeamsTx(ctx, tx, data, opts)
	assert.Error(t, err)
}

func TestBackupRepo_ImportUsersTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "import_user")
	f.AddUserToTeam(t, user.ID, team.ID)

	data := &entity.BackupData{
		Users: []entity.UserExport{
			{ID: user.ID, Username: "updated_user", Email: user.Email, Role: user.Role, TeamID: &team.ID},
		},
	}
	opts := entity.ImportOptions{ConflictMode: entity.ConflictModeOverwrite}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportUsersTx(ctx, tx, data, opts)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	got, err := f.UserRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated_user", got.Username)
}

func TestBackupRepo_ImportUsersTx_Error_InvalidTeamID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "import_user_err")
	badTeamID := uuid.New()
	data := &entity.BackupData{
		Users: []entity.UserExport{
			{ID: user.ID, Username: user.Username, Email: user.Email, Role: user.Role, TeamID: &badTeamID},
		},
	}
	opts := entity.ImportOptions{ConflictMode: entity.ConflictModeOverwrite}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportUsersTx(ctx, tx, data, opts)
	assert.Error(t, err)
}

func TestBackupRepo_ImportAwardsTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "award_import")
	awardID := uuid.New()
	data := &entity.BackupData{
		Awards: []entity.Award{
			{ID: awardID, TeamID: team.ID, Value: 100, Description: "Bonus", CreatedAt: time.Now()},
		},
	}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportAwardsTx(ctx, tx, data)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	awards, err := f.AwardRepo.GetByTeamID(ctx, team.ID)
	require.NoError(t, err)
	assert.Len(t, awards, 1)
	assert.Equal(t, 100, awards[0].Value)
}

func TestBackupRepo_ImportAwardsTx_Error_InvalidTeamID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	data := &entity.BackupData{
		Awards: []entity.Award{
			{ID: uuid.New(), TeamID: uuid.New(), Value: 100, Description: "Bad", CreatedAt: time.Now()},
		},
	}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportAwardsTx(ctx, tx, data)
	assert.Error(t, err)
}

func TestBackupRepo_ImportSolvesTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "solve_import")
	challenge := f.CreateChallenge(t, "SolveImport", 100)
	solveID := uuid.New()
	data := &entity.BackupData{
		Solves: []entity.Solve{
			{ID: solveID, UserID: user.ID, TeamID: team.ID, ChallengeID: challenge.ID, SolvedAt: time.Now()},
		},
	}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportSolvesTx(ctx, tx, data)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	got, err := f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.UserID)
}

func TestBackupRepo_ImportSolvesTx_Error_InvalidTeamID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "solve_err_user")
	challenge := f.CreateChallenge(t, "SolveErr", 100)
	data := &entity.BackupData{
		Solves: []entity.Solve{
			{ID: uuid.New(), UserID: user.ID, TeamID: uuid.New(), ChallengeID: challenge.ID, SolvedAt: time.Now()},
		},
	}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportSolvesTx(ctx, tx, data)
	assert.Error(t, err)
}

func TestBackupRepo_ImportFileMetadataTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "FileImport", 100)
	fileID := uuid.New()
	data := &entity.BackupData{
		Files: []entity.File{
			{ID: fileID, Type: entity.FileTypeChallenge, ChallengeID: challenge.ID, Location: "test/path", Filename: "file.txt", Size: 100, SHA256: "abc", CreatedAt: time.Now()},
		},
	}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportFileMetadataTx(ctx, tx, data)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	got, err := f.FileRepo.GetByID(ctx, fileID)
	require.NoError(t, err)
	assert.Equal(t, "file.txt", got.Filename)
}

func TestBackupRepo_ImportFileMetadataTx_Error_InvalidChallengeID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	data := &entity.BackupData{
		Files: []entity.File{
			{ID: uuid.New(), Type: entity.FileTypeChallenge, ChallengeID: uuid.New(), Location: "x", Filename: "f", Size: 0, SHA256: "x", CreatedAt: time.Now()},
		},
	}

	tx, err := f.TxRepo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	err = f.BackupRepo.ImportFileMetadataTx(ctx, tx, data)
	assert.Error(t, err)
}

func TestBackupRepo_RunTransaction_FullImport_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "full_import")
	challenge := f.CreateChallenge(t, "FullImport", 200)
	f.AddUserToTeam(t, user.ID, team.ID)

	data := f.NewMinimalBackupData(t)
	comp, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)
	data.Competition = comp
	data.Challenges = []entity.ChallengeExport{
		{Challenge: *challenge, Hints: []entity.Hint{}},
	}
	data.Teams = []entity.TeamExport{
		{Team: *team, MemberIDs: []uuid.UUID{user.ID}},
	}
	data.Users = []entity.UserExport{
		{ID: user.ID, Username: user.Username, Email: user.Email, Role: user.Role, TeamID: &team.ID},
	}
	opts := entity.ImportOptions{ConflictMode: entity.ConflictModeOverwrite}

	err = f.TxRepo.RunTransaction(ctx, func(txCtx context.Context, tx pgx.Tx) error {
		if err := f.BackupRepo.ImportCompetitionTx(txCtx, tx, data.Competition); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportChallengesTx(txCtx, tx, data); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportTeamsTx(txCtx, tx, data, opts); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportUsersTx(txCtx, tx, data, opts); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportAwardsTx(txCtx, tx, data); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportSolvesTx(txCtx, tx, data); err != nil {
			return err
		}
		return f.BackupRepo.ImportFileMetadataTx(txCtx, tx, data)
	})

	require.NoError(t, err)

	gotComp, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, data.Competition.Name, gotComp.Name)
}

func TestBackupRepo_RunTransaction_EraseAndImportChallenges_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "EraseImport", 300)
	challengeID := challenge.ID

	data := f.NewMinimalBackupData(t)
	data.Competition = &entity.Competition{ID: 1, Name: "Restored", Mode: "flexible"}
	data.Challenges = []entity.ChallengeExport{
		{Challenge: entity.Challenge{ID: challengeID, Title: "Restored Chall", Description: challenge.Description, Category: challenge.Category, Points: 400, FlagHash: challenge.FlagHash, InitialValue: 400, MinValue: 400, Decay: 0, SolveCount: 0}, Hints: []entity.Hint{}},
	}

	err := f.TxRepo.RunTransaction(ctx, func(txCtx context.Context, tx pgx.Tx) error {
		if err := f.BackupRepo.EraseAllTablesTx(txCtx, tx); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportCompetitionTx(txCtx, tx, data.Competition); err != nil {
			return err
		}
		return f.BackupRepo.ImportChallengesTx(txCtx, tx, data)
	})

	require.NoError(t, err)

	got, err := f.ChallengeRepo.GetByID(ctx, challengeID)
	require.NoError(t, err)
	assert.Equal(t, "Restored Chall", got.Title)
	assert.Equal(t, 400, got.Points)
}
