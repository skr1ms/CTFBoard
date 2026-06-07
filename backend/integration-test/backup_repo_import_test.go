package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestBackupRepo_ImportCompetitionTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	t.Cleanup(func() { f.ResetCompetition(t) })

	comp := f.ActiveCompetition(t, "Updated CTF")

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportCompetition(txCtx, comp)
	})
	require.NoError(t, err)

	got, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Updated CTF", got.Name)
}

func TestBackupRepo_ImportCompetitionTx_NilCompetition(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportCompetition(txCtx, nil)
	})
	require.NoError(t, err)
}

func TestBackupRepo_ImportChallengesTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challengeID := uuid.New()
	nextChallengeID := uuid.New()
	data := &domain.BackupData{
		Challenges: []domain.ChallengeExport{
			{
				Challenge: domain.Challenge{
					ID:              challengeID,
					Title:           "Backup Chall",
					Description:     "Desc",
					Category:        "Web",
					Points:          150,
					FlagHash:        "hash",
					Attribution:     "Author",
					NextChallengeID: &nextChallengeID,
					InitialValue:    150,
					MinValue:        150,
					Decay:           0,
				},
				Attribution: "Author",
				NextID:      &nextChallengeID,
				Hints:       []domain.Hint{},
			},
			{
				Challenge: domain.Challenge{
					ID:           nextChallengeID,
					Title:        "Next Backup Chall",
					Description:  "Next Desc",
					Category:     "Web",
					Points:       100,
					FlagHash:     "next_hash",
					InitialValue: 100,
					MinValue:     100,
					Decay:        0,
				},
				Hints: []domain.Hint{},
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
	assert.Equal(t, "Author", got.Attribution)
	require.NotNil(t, got.NextChallengeID)
	assert.Equal(t, nextChallengeID, *got.NextChallengeID)
}

func TestBackupRepo_ImportChallengesTx_Error_InvalidHintChallengeID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challengeID := uuid.New()
	nonexistentChallengeID := uuid.New()
	data := &domain.BackupData{
		Challenges: []domain.ChallengeExport{
			{
				Challenge: domain.Challenge{ID: challengeID, Title: "Ch", Description: "D", Category: "W", Points: 100, FlagHash: "h", InitialValue: 100, MinValue: 100, Decay: 0},
				Hints: []domain.Hint{
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
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, _ := f.CreateUserWithTeam(t, "import_team_user")
	teamID := uuid.New()
	data := &domain.BackupData{
		Teams: []domain.TeamExport{
			{
				Team: domain.Team{
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
	opts := domain.ImportOptions{ConflictMode: domain.ConflictModeOverwrite}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportTeams(txCtx, data, opts)
	})
	require.NoError(t, err)
	got, err := f.TeamRepo.GetByID(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, "Imported Team", got.Name)
}

func TestBackupRepo_ImportTeamsTx_Error_InvalidCaptainID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	teamID := uuid.New()
	data := &domain.BackupData{
		Teams: []domain.TeamExport{
			{
				Team: domain.Team{
					ID:          teamID,
					Name:        "Bad Team",
					CaptainID:   uuid.New(),
					InviteToken: uuid.New(),
					CreatedAt:   time.Now(),
				},
			},
		},
	}
	opts := domain.ImportOptions{ConflictMode: domain.ConflictModeOverwrite}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportTeams(txCtx, data, opts)
	})
	assert.Error(t, err)
}

func TestBackupRepo_ImportUsersTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "import_user")
	f.AddUserToTeam(t, user.ID, team.ID)

	data := &domain.BackupData{
		Users: []domain.UserExport{
			{ID: user.ID, Username: "updated_user", Email: user.Email, Role: string(user.Role), TeamID: &team.ID},
		},
	}
	opts := domain.ImportOptions{ConflictMode: domain.ConflictModeOverwrite}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportUsers(txCtx, data, opts)
	})
	require.NoError(t, err)
	got, err := f.UserRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated_user", got.Username)
}

func TestBackupRepo_ImportUsersTx_Error_InvalidTeamID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "import_user_err")
	badTeamID := uuid.New()
	data := &domain.BackupData{
		Users: []domain.UserExport{
			{ID: user.ID, Username: user.Username, Email: user.Email, Role: string(user.Role), TeamID: &badTeamID},
		},
	}
	opts := domain.ImportOptions{ConflictMode: domain.ConflictModeOverwrite}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		err := f.BackupRepo.ImportUsers(txCtx, data, opts)
		if err != nil {
			return err
		}

		return f.BackupRepo.UpdateUserTeamIDs(txCtx, data)
	})
	assert.Error(t, err)
}

func TestBackupRepo_ImportAwardsTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "award_import")
	awardID := uuid.New()
	data := &domain.BackupData{
		Awards: []domain.Award{
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
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	data := &domain.BackupData{
		Awards: []domain.Award{
			{ID: uuid.New(), TeamID: uuid.New(), Value: 100, Description: "Bad", CreatedAt: time.Now()},
		},
	}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportAwards(txCtx, data)
	})
	assert.Error(t, err)
}

func TestBackupRepo_ImportSolvesTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "solve_import")
	challenge := f.CreateChallenge(t, "SolveImport", 100)
	solveID := uuid.New()
	data := &domain.BackupData{
		Solves: []domain.Solve{
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
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "solve_err_user")
	challenge := f.CreateChallenge(t, "SolveErr", 100)
	data := &domain.BackupData{
		Solves: []domain.Solve{
			{ID: uuid.New(), UserID: user.ID, TeamID: uuid.New(), ChallengeID: challenge.ID, SolvedAt: time.Now()},
		},
	}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportSolves(txCtx, data)
	})
	assert.Error(t, err)
}

func TestBackupRepo_ImportFileMetadataTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "FileImport", 100)
	fileID := uuid.New()
	data := &domain.BackupData{
		Files: []domain.File{
			{ID: fileID, Type: domain.FileTypeChallenge, ChallengeID: &challenge.ID, Location: "test/path", Filename: "file.txt", Size: 100, SHA256: "abc", CreatedAt: time.Now()},
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
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	data := &domain.BackupData{
		Files: []domain.File{
			{ID: uuid.New(), Type: domain.FileTypeChallenge, ChallengeID: func() *uuid.UUID {
				id := uuid.New()

				return &id
			}(), Location: "x", Filename: "f", Size: 0, SHA256: "x", CreatedAt: time.Now()},
		},
	}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportFileMetadata(txCtx, data)
	})
	assert.Error(t, err)
}

func TestBackupRepo_ImportFieldsTx_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	fieldID := uuid.New()
	jsonFieldID := uuid.New()
	data := &domain.BackupData{
		Fields: []domain.Field{
			{
				ID:          fieldID,
				Name:        "division",
				Description: "Player division",
				FieldType:   domain.FieldTypeSelect,
				EntityType:  domain.EntityTypeUser,
				Required:    true,
				Public:      true,
				Editable:    true,
				Options:     []string{"junior", "senior"},
				OrderIndex:  3,
				CreatedAt:   time.Now(),
			},
			{
				ID:          jsonFieldID,
				Name:        "metadata",
				Description: "Structured metadata",
				FieldType:   domain.FieldTypeJSON,
				EntityType:  domain.EntityTypeTeam,
				Public:      true,
				Editable:    true,
				OrderIndex:  4,
				CreatedAt:   time.Now(),
			},
		},
	}

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.BackupRepo.ImportFields(txCtx, data)
	})
	require.NoError(t, err)

	got, err := f.FieldRepo.GetByID(ctx, fieldID)
	require.NoError(t, err)
	assert.Equal(t, "division", got.Name)
	assert.Equal(t, "Player division", got.Description)
	assert.True(t, got.Required)
	assert.True(t, got.Public)
	assert.True(t, got.Editable)
	assert.Equal(t, []string{"junior", "senior"}, got.Options)
	assert.Equal(t, 3, got.OrderIndex)

	jsonField, err := f.FieldRepo.GetByID(ctx, jsonFieldID)
	require.NoError(t, err)
	assert.Equal(t, domain.FieldTypeJSON, jsonField.FieldType)
	assert.Equal(t, domain.EntityTypeTeam, jsonField.EntityType)
	assert.Equal(t, "Structured metadata", jsonField.Description)
}
