package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestAuditLogRepo_Create_Success(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	user := f.CreateUser(t, uuid.New().String())

	auditLog := &entity.AuditLog{
		UserID:     &user.ID,
		Action:     "create",
		EntityType: entity.RoleUser,
		EntityID:   user.ID.String(),
		IP:         "127.0.0.1",
		Details:    map[string]any{"foo": "bar"},
	}

	err := f.AuditLogRepo.Create(context.Background(), auditLog)
	require.NoError(t, err)

	var count int
	err = f.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM audit_logs WHERE user_id=$1", user.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestAuditLogRepo_Create_Error_InvalidUUID(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	nonExistentuuid := uuid.New()

	auditLog := &entity.AuditLog{
		UserID:     &nonExistentuuid,
		Action:     "create",
		EntityType: entity.RoleUser,
		EntityID:   "something",
		IP:         "127.0.0.1",
	}

	err := f.AuditLogRepo.Create(context.Background(), auditLog)
	require.Error(t, err)
}
