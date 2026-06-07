package backup

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	logMock "github.com/wahrwelt-kit/go-logkit/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	backupMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup/mock"
)

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
