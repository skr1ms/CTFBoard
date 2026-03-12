package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
)

func TestBackupRepo_EraseAllTablesTx_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "erase_user")
	challenge := f.CreateChallenge(t, "erase_chall", 100)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseAllTables(txCtx)
	})
	require.NoError(t, err)

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
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseAllTables(txCtx)
	})
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportCompetition(txCtx, comp)
	})
	require.NoError(t, err)

	got, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Updated CTF", got.Name)
}

func TestBackupRepo_ImportCompetitionTx_NilCompetition(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportCompetition(txCtx, nil)
	})
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportChallenges(txCtx, data)
	})
	require.NoError(t, err)
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportChallenges(txCtx, data)
	})
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportTeams(txCtx, data, opts)
	})
	require.NoError(t, err)
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportTeams(txCtx, data, opts)
	})
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
			{ID: user.ID, Username: "updated_user", Email: user.Email, Role: string(user.Role), TeamID: &team.ID},
		},
	}
	opts := entity.ImportOptions{ConflictMode: entity.ConflictModeOverwrite}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportUsers(txCtx, data, opts)
	})
	require.NoError(t, err)
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
			{ID: user.ID, Username: user.Username, Email: user.Email, Role: string(user.Role), TeamID: &badTeamID},
		},
	}
	opts := entity.ImportOptions{ConflictMode: entity.ConflictModeOverwrite}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportUsers(txCtx, data, opts)
	})
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportAwards(txCtx, data)
	})
	require.NoError(t, err)
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportAwards(txCtx, data)
	})
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportSolves(txCtx, data)
	})
	require.NoError(t, err)
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportSolves(txCtx, data)
	})
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportFileMetadata(txCtx, data)
	})
	require.NoError(t, err)
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportFileMetadata(txCtx, data)
	})
	assert.Error(t, err)
}

func TestBackupRepo_TM_Run_FullImport_Success(t *testing.T) {
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
		{ID: user.ID, Username: user.Username, Email: user.Email, Role: string(user.Role), TeamID: &team.ID},
	}
	opts := entity.ImportOptions{ConflictMode: entity.ConflictModeOverwrite}

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		if err := f.BackupRepo.ImportCompetition(txCtx, data.Competition); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportChallenges(txCtx, data); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportTeams(txCtx, data, opts); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportUsers(txCtx, data, opts); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportAwards(txCtx, data); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportSolves(txCtx, data); err != nil {
			return err
		}
		return f.BackupRepo.ImportFileMetadata(txCtx, data)
	})

	require.NoError(t, err)

	gotComp, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, data.Competition.Name, gotComp.Name)
}

func TestBackupRepo_TM_Run_EraseAndImportChallenges_Success(t *testing.T) {
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

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		if err := f.BackupRepo.EraseAllTables(txCtx); err != nil {
			return err
		}
		if err := f.BackupRepo.ImportCompetition(txCtx, data.Competition); err != nil {
			return err
		}
		return f.BackupRepo.ImportChallenges(txCtx, data)
	})

	require.NoError(t, err)

	got, err := f.ChallengeRepo.GetByID(ctx, challengeID)
	require.NoError(t, err)
	assert.Equal(t, "Restored Chall", got.Title)
	assert.Equal(t, 400, got.Points)
}

func TestBackupRepo_EraseTables_DisallowedTable_ReturnsError(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseTables(txCtx, []string{"solves", "evil_table"})
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not allowed")
	assert.Contains(t, err.Error(), "evil_table")
}

func TestBackupRepo_EraseTables_AllowedSubset_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "erase_subset")
	challenge := f.CreateChallenge(t, "erase_subset_ch", 100)
	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseTables(txCtx, []string{"solves"})
	})
	require.NoError(t, err)

	_, err = f.SolveRepo.GetByTeamAndChallenge(ctx, team.ID, challenge.ID)
	assert.Error(t, err)
	_, err = f.UserRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	_, err = f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
}

func TestBackupRepo_EraseTables_EmptyTables_NoOp(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseTables(txCtx, nil)
	})
	require.NoError(t, err)

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.EraseTables(txCtx, []string{})
	})
	require.NoError(t, err)
}
