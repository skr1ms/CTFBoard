package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	logMock "github.com/wahrwelt-kit/go-logkit/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	backupMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup/mock"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func TestNewBackupUseCase(t *testing.T) {
	t.Parallel()
	deps := BackupDeps{
		Logger: logMock.NewMockLogger(t),
	}
	uc := NewBackupUseCase(deps)
	assert.NotNil(t, uc)
}

func TestBackupUseCase_Export_Success(t *testing.T) {
	t.Parallel()
	compRepo := backupMock.NewMockCompetitionRepository(t)
	challRepo := backupMock.NewMockChallengeRepository(t)
	storage := backupMock.NewMockStorageProvider(t)
	log := logMock.NewMockLogger(t)

	comp := &domain.Competition{Name: "Test", Mode: "flexible"}
	compRepo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()
	challRepo.EXPECT().GetAllForBackup(mock.Anything).Return([]*repo.ChallengeWithSolved{}, nil).Once()
	challRepo.EXPECT().GetAllRequirementPairs(mock.Anything).Return([]*domain.ChallengeRequirementPair{}, nil).Maybe()
	challRepo.EXPECT().GetAllSolutions(mock.Anything).Return([]*domain.SolutionBackup{}, nil).Maybe()

	deps := BackupDeps{
		CompetitionRepo: compRepo,
		ChallengeRepo:   challRepo,
		HintRepo:        nil,
		TeamRepo:        nil,
		UserRepo:        nil,
		AwardRepo:       nil,
		SolveRepo:       nil,
		FileRepo:        nil,
		BackupRepo:      nil,
		Storage:         storage,
		TM:              nil,
		Logger:          log,
	}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	opts := domain.ExportOptions{}

	data, err := uc.Export(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, domain.BackupVersion, data.Version)
	assert.NotZero(t, data.ExportedAt)
	assert.Equal(t, comp, data.Competition)
	assert.Empty(t, data.Challenges)
}

func TestBackupUseCase_Export_CompetitionRepoError(t *testing.T) {
	t.Parallel()
	compRepo := backupMock.NewMockCompetitionRepository(t)
	challRepo := backupMock.NewMockChallengeRepository(t)
	log := logMock.NewMockLogger(t)

	compRepo.EXPECT().Get(mock.Anything).Return(nil, errors.New("db error")).Once()
	challRepo.EXPECT().GetAllForBackup(mock.Anything).Return([]*repo.ChallengeWithSolved{}, nil).Maybe()
	challRepo.EXPECT().GetAllRequirementPairs(mock.Anything).Return([]*domain.ChallengeRequirementPair{}, nil).Maybe()
	challRepo.EXPECT().GetAllSolutions(mock.Anything).Return([]*domain.SolutionBackup{}, nil).Maybe()

	storage := backupMock.NewMockStorageProvider(t)

	deps := BackupDeps{
		CompetitionRepo: compRepo,
		ChallengeRepo:   challRepo,
		Logger:          log,
		Storage:         storage,
	}
	uc := NewBackupUseCase(deps)

	_, err := uc.Export(context.Background(), domain.ExportOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BackupUseCase - Export")
}

func TestBackupUseCase_ExportZIP_Success(t *testing.T) {
	t.Parallel()
	compRepo := backupMock.NewMockCompetitionRepository(t)
	challRepo := backupMock.NewMockChallengeRepository(t)
	storage := backupMock.NewMockStorageProvider(t)
	log := logMock.NewMockLogger(t)

	comp := &domain.Competition{Name: "Test", Mode: "flexible"}
	compRepo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()
	challRepo.EXPECT().GetAllForBackup(mock.Anything).Return([]*repo.ChallengeWithSolved{}, nil).Once()
	challRepo.EXPECT().GetAllRequirementPairs(mock.Anything).Return([]*domain.ChallengeRequirementPair{}, nil).Maybe()
	challRepo.EXPECT().GetAllSolutions(mock.Anything).Return([]*domain.SolutionBackup{}, nil).Maybe()
	log.EXPECT().Info(mock.Anything, mock.Anything).Maybe()

	deps := BackupDeps{
		CompetitionRepo: compRepo,
		ChallengeRepo:   challRepo,
		Storage:         storage,
		Logger:          log,
	}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	rc, err := uc.ExportZIP(ctx, domain.ExportOptions{})
	require.NoError(t, err)

	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), 4)
	assert.Equal(t, []byte("PK")[0], data[0])
	assert.Equal(t, []byte("PK")[1], data[1])
}

func TestBackupUseCase_ExportZIP_Error(t *testing.T) {
	t.Parallel()
	compRepo := backupMock.NewMockCompetitionRepository(t)
	challRepo := backupMock.NewMockChallengeRepository(t)
	storage := backupMock.NewMockStorageProvider(t)
	log := logMock.NewMockLogger(t)

	compRepo.EXPECT().Get(mock.Anything).Return(nil, errors.New("db error")).Once()
	challRepo.EXPECT().GetAllForBackup(mock.Anything).Return([]*repo.ChallengeWithSolved{}, nil).Maybe()
	challRepo.EXPECT().GetAllRequirementPairs(mock.Anything).Return([]*domain.ChallengeRequirementPair{}, nil).Maybe()
	challRepo.EXPECT().GetAllSolutions(mock.Anything).Return([]*domain.SolutionBackup{}, nil).Maybe()

	deps := BackupDeps{
		CompetitionRepo: compRepo,
		ChallengeRepo:   challRepo,
		Storage:         storage,
		Logger:          log,
	}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	rc, err := uc.ExportZIP(ctx, domain.ExportOptions{})
	require.NoError(t, err)

	defer rc.Close()

	_, err = io.ReadAll(rc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestBackupUseCase_ImportZIP_Success(t *testing.T) {
	t.Parallel()
	tm := backupMock.NewMockTransactionManager(t)
	backupRepo := backupMock.NewMockBackupRepository(t)
	log := logMock.NewMockLogger(t)

	backupData := &domain.BackupData{
		Version:     domain.BackupVersion,
		ExportedAt:  time.Now().UTC(),
		Competition: &domain.Competition{Name: "Test", Mode: "flexible"},
	}
	zipBuf := new(bytes.Buffer)
	zw := zip.NewWriter(zipBuf)
	w, err := zw.Create("backup.json")
	require.NoError(t, err)
	require.NoError(t, json.NewEncoder(w).Encode(backupData))
	require.NoError(t, zw.Close())

	tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	backupRepo.EXPECT().ImportCompetition(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportTags(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportChallenges(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportChallengeTags(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportBrackets(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportUsers(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportTeams(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().UpdateUserTeamIDs(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportAwards(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportSolves(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportHintUnlocks(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportFileMetadata(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportChallengeRequirements(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportSolutions(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportRatings(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportComments(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportFields(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportFieldValues(mock.Anything, mock.Anything).Return(nil).Once()
	log.EXPECT().Info(mock.Anything, mock.Anything).Maybe()

	deps := BackupDeps{
		TM:         tm,
		BackupRepo: backupRepo,
		Logger:     log,
	}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	r := bytes.NewReader(zipBuf.Bytes())
	readerAt := io.NewSectionReader(r, 0, int64(zipBuf.Len()))

	result, err := uc.ImportZIP(ctx, readerAt, int64(zipBuf.Len()), domain.ImportOptions{})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestBackupUseCase_ImportZIP_Error_InvalidZIP(t *testing.T) {
	t.Parallel()
	log := logMock.NewMockLogger(t)

	deps := BackupDeps{Logger: log}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	invalidZIP := bytes.NewReader([]byte("not a zip file"))
	readerAt := io.NewSectionReader(invalidZIP, 0, 13)

	_, err := uc.ImportZIP(ctx, readerAt, 13, domain.ImportOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BackupUseCase - ImportZIP")
}

func TestBackupUseCase_Reset_Success(t *testing.T) {
	t.Parallel()
	tm := backupMock.NewMockTransactionManager(t)
	backupRepo := backupMock.NewMockBackupRepository(t)
	log := logMock.NewMockLogger(t)

	tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	backupRepo.EXPECT().EraseTables(mock.Anything, mock.AnythingOfType("[]string")).Return(nil).Once()

	deps := BackupDeps{
		TM:         tm,
		BackupRepo: backupRepo,
		Logger:     log,
	}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	opts := domain.AdminResetOptions{Submissions: true}

	err := uc.Reset(ctx, opts)

	require.NoError(t, err)
}

func TestBackupUseCase_Reset_Error(t *testing.T) {
	t.Parallel()
	tm := backupMock.NewMockTransactionManager(t)
	log := logMock.NewMockLogger(t)

	tm.EXPECT().Run(mock.Anything, mock.Anything).Return(errors.New("tx failed")).Once()

	deps := BackupDeps{
		TM:     tm,
		Logger: log,
	}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	opts := domain.AdminResetOptions{Submissions: true}

	err := uc.Reset(ctx, opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tx failed")
}

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

func TestBackupUseCase_ExportCSV_Error_UnknownTable(t *testing.T) {
	t.Parallel()
	log := logMock.NewMockLogger(t)

	deps := BackupDeps{Logger: log}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()

	_, err := uc.ExportCSV(ctx, "unknown_table")

	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrBackupTableUnsupported)
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
	assert.ErrorIs(t, err, httperr.ErrBackupTableUnsupported)
}
