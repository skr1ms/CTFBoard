package backup

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	logMock "github.com/wahrwelt-kit/go-logkit/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	backupMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup/mock"
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
	storage := backupMock.NewMockBackupStorage(t)
	log := logMock.NewMockLogger(t)

	comp := &domain.Competition{Name: "Test", Mode: "teams_only"}
	nextID := uuid.New()
	challengeID := uuid.New()

	compRepo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()
	challRepo.EXPECT().GetAllForBackup(mock.Anything).Return([]*repo.ChallengeWithSolved{
		{
			Challenge: &domain.Challenge{
				ID:              challengeID,
				Title:           "Challenge",
				Attribution:     "Author",
				NextChallengeID: &nextID,
			},
		},
	}, nil).Once()
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
	require.Len(t, data.Challenges, 1)
	assert.Equal(t, "Author", data.Challenges[0].Attribution)
	require.NotNil(t, data.Challenges[0].NextID)
	assert.Equal(t, nextID, *data.Challenges[0].NextID)
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

	storage := backupMock.NewMockBackupStorage(t)

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
	storage := backupMock.NewMockBackupStorage(t)
	log := logMock.NewMockLogger(t)

	comp := &domain.Competition{Name: "Test", Mode: "teams_only"}
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
	storage := backupMock.NewMockBackupStorage(t)
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

func TestBackupUseCase_ExportZIP_CloseCancelsWorker(t *testing.T) {
	t.Parallel()
	compRepo := backupMock.NewMockCompetitionRepository(t)
	challRepo := backupMock.NewMockChallengeRepository(t)
	storage := backupMock.NewMockBackupStorage(t)
	log := logMock.NewMockLogger(t)

	started := make(chan struct{})

	compRepo.EXPECT().Get(mock.Anything).RunAndReturn(func(ctx context.Context) (*domain.Competition, error) {
		close(started)
		<-ctx.Done()

		return nil, ctx.Err()
	}).Once()
	challRepo.EXPECT().GetAllForBackup(mock.Anything).Return([]*repo.ChallengeWithSolved{}, nil).Maybe()
	challRepo.EXPECT().GetAllRequirementPairs(mock.Anything).Return([]*domain.ChallengeRequirementPair{}, nil).Maybe()
	challRepo.EXPECT().GetAllSolutions(mock.Anything).Return([]*domain.SolutionBackup{}, nil).Maybe()

	uc := NewBackupUseCase(BackupDeps{
		CompetitionRepo: compRepo,
		ChallengeRepo:   challRepo,
		Storage:         storage,
		Logger:          log,
	})

	rc, err := uc.ExportZIP(context.Background(), domain.ExportOptions{})
	require.NoError(t, err)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("export worker did not start")
	}

	closed := make(chan error, 1)

	go func() {
		closed <- rc.Close()
	}()

	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("ExportZIP Close did not wait-boundedly for worker shutdown")
	}
}
