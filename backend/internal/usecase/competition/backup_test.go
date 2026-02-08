package competition

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type stubStorage struct{}

func (stubStorage) Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error {
	return nil
}

func (stubStorage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, nil
}

func (stubStorage) Delete(ctx context.Context, path string) error {
	return nil
}

func (stubStorage) GetPresignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return "", nil
}

func TestNewBackupUseCase(t *testing.T) {
	deps := BackupDeps{
		Logger: mocks.NewMockLogger(t),
	}
	uc := NewBackupUseCase(deps)
	assert.NotNil(t, uc)
}

func TestBackupUseCase_Export_Success(t *testing.T) {
	compRepo := mocks.NewMockCompetitionRepository(t)
	challRepo := mocks.NewMockChallengeRepository(t)
	logger := mocks.NewMockLogger(t)

	comp := &entity.Competition{Name: "Test", Mode: "flexible"}
	compRepo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()
	challRepo.EXPECT().GetAll(mock.Anything, mock.Anything, mock.Anything).Return([]*repo.ChallengeWithSolved{}, nil).Once()

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
		Storage:         stubStorage{},
		TxRepo:          nil,
		Logger:          logger,
	}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	opts := entity.ExportOptions{}

	data, err := uc.Export(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, entity.BackupVersion, data.Version)
	assert.NotZero(t, data.ExportedAt)
	assert.Equal(t, comp, data.Competition)
	assert.Empty(t, data.Challenges)
}

func TestBackupUseCase_Export_CompetitionRepoError(t *testing.T) {
	compRepo := mocks.NewMockCompetitionRepository(t)
	challRepo := mocks.NewMockChallengeRepository(t)
	logger := mocks.NewMockLogger(t)

	compRepo.EXPECT().Get(mock.Anything).Return(nil, errors.New("db error")).Once()
	challRepo.EXPECT().GetAll(mock.Anything, mock.Anything, mock.Anything).Return([]*repo.ChallengeWithSolved{}, nil).Maybe()

	deps := BackupDeps{
		CompetitionRepo: compRepo,
		ChallengeRepo:   challRepo,
		Logger:          logger,
		Storage:         stubStorage{},
	}
	uc := NewBackupUseCase(deps)

	_, err := uc.Export(context.Background(), entity.ExportOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BackupUseCase - Export")
}
