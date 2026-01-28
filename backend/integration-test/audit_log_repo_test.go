package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestAuditLogRepo_Create_Success(t *testing.T) {
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	user := f.CreateUser(t, uuid.New().String())

	auditLog := &entity.AuditLog{
		UserId:     &user.Id,
		Action:     "create",
		EntityType: entity.RoleUser,
		EntityId:   user.Id.String(),
		IP:         "127.0.0.1",
		Details:    map[string]any{"foo": "bar"},
	}

	err := f.AuditLogRepo.Create(context.Background(), auditLog)
	require.NoError(t, err)

	var count int
	err = f.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM audit_logs WHERE user_id=$1", user.Id).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestAuditLogRepo_Create_Error_InvalidUUID(t *testing.T) {
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	nonExistentUUID := uuid.New()

	auditLog := &entity.AuditLog{
		UserId:     &nonExistentUUID,
		Action:     "create",
		EntityType: entity.RoleUser,
		EntityId:   "something",
		IP:         "127.0.0.1",
	}

	err := f.AuditLogRepo.Create(context.Background(), auditLog)
	require.Error(t, err)
}
