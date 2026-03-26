package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestAuditLogTx_SettingsUpdate_CommitsBoth(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)

	ctx := context.Background()

	actor := f.CreateUser(t, "audit_commit")
	original, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)

	original.AppName = "AuditTxCommit"
	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		innerErr := f.SettingsRepo.Update(txCtx, original)
		if innerErr != nil {
			return innerErr
		}

		return f.AuditLogRepo.Create(txCtx, &domain.AuditLog{
			UserID:     &actor.ID,
			Action:     domain.AuditActionUpdate,
			EntityType: domain.AuditEntityAppSettings,
			EntityID:   "settings",
			IP:         "127.0.0.1",
			Details:    map[string]any{"app_name": "AuditTxCommit"},
		})
	})
	require.NoError(t, err)

	updated, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "AuditTxCommit", updated.AppName)

	var count int

	err = f.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs WHERE user_id = $1", actor.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "audit log entry should be persisted")
}

func TestAuditLogTx_SettingsUpdate_RollbackOnForcedError(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)

	ctx := context.Background()

	actor := f.CreateUser(t, "audit_rb")
	original, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)

	originalName := original.AppName

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		original.AppName = "ShouldNeverPersist"

		innerErr := f.SettingsRepo.Update(txCtx, original)
		if innerErr != nil {
			return innerErr
		}

		innerErr = f.AuditLogRepo.Create(txCtx, &domain.AuditLog{
			UserID:     &actor.ID,
			Action:     domain.AuditActionUpdate,
			EntityType: domain.AuditEntityAppSettings,
			EntityID:   "settings",
			IP:         "127.0.0.1",
		})
		if innerErr != nil {
			return innerErr
		}

		return errors.New("forced rollback after both writes")
	})
	require.Error(t, err)

	current, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, originalName, current.AppName, "settings must be rolled back")

	var count int

	err = f.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs WHERE user_id = $1", actor.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "audit log must be rolled back")
}

func TestAuditLogTx_SettingsUpdate_RollbackOnInvalidAuditLog(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)

	ctx := context.Background()

	original, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)

	originalName := original.AppName

	nonExistentUserID := uuid.New()
	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		original.AppName = "FailAuditLog"

		innerErr := f.SettingsRepo.Update(txCtx, original)
		if innerErr != nil {
			return innerErr
		}

		return f.AuditLogRepo.Create(txCtx, &domain.AuditLog{
			UserID:     &nonExistentUserID,
			Action:     domain.AuditActionUpdate,
			EntityType: domain.AuditEntityAppSettings,
			EntityID:   "settings",
			IP:         "127.0.0.1",
		})
	})
	require.Error(t, err, "audit log FK violation must propagate")

	current, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, originalName, current.AppName, "settings must be rolled back when audit log fails")
}
