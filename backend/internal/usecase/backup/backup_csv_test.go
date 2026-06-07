package backup

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	logMock "github.com/wahrwelt-kit/go-logkit/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	backupMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup/mock"
)

func TestBackupUseCase_ExportCSV_Success(t *testing.T) {
	t.Parallel()
	userRepo := backupMock.NewMockUserRepository(t)
	log := logMock.NewMockLogger(t)

	userRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.User{}, nil).Once()

	deps := BackupDeps{
		UserRepo: userRepo,
		Logger:   log,
	}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()

	data, err := uc.ExportCSV(ctx, "users")

	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Contains(t, string(data), "id")
	assert.Contains(t, string(data), "username")
}

func TestBackupUseCase_ExportCSV_ChallengesIncludesMetadata(t *testing.T) {
	t.Parallel()
	challengeRepo := backupMock.NewMockChallengeRepository(t)
	log := logMock.NewMockLogger(t)

	nextID := uuid.New()
	challengeRepo.EXPECT().GetAllForBackup(mock.Anything).Return([]*repo.ChallengeWithSolved{
		{
			Challenge: &domain.Challenge{
				ID:              uuid.New(),
				Title:           "Challenge",
				Attribution:     "Author",
				NextChallengeID: &nextID,
			},
		},
	}, nil).Once()

	uc := NewBackupUseCase(BackupDeps{
		ChallengeRepo: challengeRepo,
		Logger:        log,
	})

	data, err := uc.ExportCSV(context.Background(), "challenges")

	require.NoError(t, err)

	csv := string(data)
	assert.Contains(t, csv, "attribution")
	assert.Contains(t, csv, "next_challenge_id")
	assert.Contains(t, csv, "Author")
	assert.Contains(t, csv, nextID.String())
}

func TestBackupUseCase_ExportCSV_Error_UnknownTable(t *testing.T) {
	t.Parallel()
	log := logMock.NewMockLogger(t)

	deps := BackupDeps{Logger: log}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()

	_, err := uc.ExportCSV(ctx, "unknown_table")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrBackupTableUnsupported)
}

func TestBackupUseCase_ImportCSV_Success(t *testing.T) {
	t.Parallel()
	tm := backupMock.NewMockTransactionManager(t)
	backupRepo := backupMock.NewMockBackupRepository(t)
	log := logMock.NewMockLogger(t)

	header := []string{"id", "username", "email", "role", "is_verified", "team_id", "created_at"}
	rows := [][]string{
		{"00000000-0000-0000-0000-000000000001", "user1", "u1@x.com", "user", "true", "", "2024-01-01T00:00:00Z"},
	}

	tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	backupRepo.EXPECT().ImportCSV(mock.Anything, "users", header, rows).Return(1, nil, nil).Once()

	deps := BackupDeps{
		TM:         tm,
		BackupRepo: backupRepo,
		Logger:     log,
	}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	csvData := []byte("id,username,email,role,is_verified,team_id,created_at\n00000000-0000-0000-0000-000000000001,user1,u1@x.com,user,true,,2024-01-01T00:00:00Z")

	result, err := uc.ImportCSV(ctx, "users", csvData)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.ImportedCount)
}

func TestBackupUseCase_ImportCSV_Error_InvalidFormat(t *testing.T) {
	t.Parallel()
	log := logMock.NewMockLogger(t)

	deps := BackupDeps{Logger: log}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	invalidCSV := []byte("id,username\n\"unclosed quote")

	_, err := uc.ImportCSV(ctx, "users", invalidCSV)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BackupUseCase - ImportCSV")
}

func TestBackupUseCase_ImportCSV_Error_UnknownTable(t *testing.T) {
	t.Parallel()
	log := logMock.NewMockLogger(t)

	deps := BackupDeps{Logger: log}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()

	_, err := uc.ImportCSV(ctx, "unknown_table", []byte("a,b\n1,2"))

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrBackupTableUnsupported)
}
