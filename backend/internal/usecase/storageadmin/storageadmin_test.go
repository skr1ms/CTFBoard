package storageadmin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type fakeStorageAdminStorage struct {
	deletedPath string
	deleteErr   error
	listPrefix  string
	listLimit   int
}

func (s *fakeStorageAdminStorage) List(_ context.Context, prefix string, limit int) ([]string, error) {
	s.listPrefix = prefix
	s.listLimit = limit

	return []string{"challenges/file.zip"}, nil
}

func (s *fakeStorageAdminStorage) Delete(_ context.Context, path string) error {
	s.deletedPath = path

	return s.deleteErr
}

type fakeStorageAdminAuditLogRepo struct {
	log *domain.AuditLog
	err error
}

func (r *fakeStorageAdminAuditLogRepo) Create(_ context.Context, log *domain.AuditLog) error {
	r.log = log

	return r.err
}

func TestUseCase_Delete_WritesAuditLog(t *testing.T) {
	t.Parallel()

	storage := &fakeStorageAdminStorage{}
	auditRepo := &fakeStorageAdminAuditLogRepo{}
	actorID := uuid.New()

	uc := NewUseCase(Deps{Storage: storage, AuditLog: auditRepo})
	err := uc.Delete(context.Background(), usecase.StorageAdminDeleteParams{
		Path:     "challenges/file.zip",
		ActorID:  actorID,
		ClientIP: "192.0.2.10",
	})

	require.NoError(t, err)
	assert.Equal(t, "challenges/file.zip", storage.deletedPath)
	require.NotNil(t, auditRepo.log)
	assert.Equal(t, &actorID, auditRepo.log.UserID)
	assert.Equal(t, domain.AuditActionDelete, auditRepo.log.Action)
	assert.Equal(t, domain.AuditEntityStorage, auditRepo.log.EntityType)
	assert.Equal(t, storageAuditEntityID, auditRepo.log.EntityID)
	assert.Equal(t, "192.0.2.10", auditRepo.log.IP)
	assert.Equal(t, "storage object delete requested", auditRepo.log.Details["message"])
	assert.Equal(t, "challenges/file.zip", auditRepo.log.Details["path"])
	assert.Equal(t, "requested", auditRepo.log.Details["status"])
}

func TestUseCase_Delete_FailsClosedWhenAuditFails(t *testing.T) {
	t.Parallel()

	storage := &fakeStorageAdminStorage{}
	auditErr := errors.New("audit unavailable")
	uc := NewUseCase(Deps{Storage: storage, AuditLog: &fakeStorageAdminAuditLogRepo{err: auditErr}})

	err := uc.Delete(context.Background(), usecase.StorageAdminDeleteParams{
		Path:    "challenges/file.zip",
		ActorID: uuid.New(),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, auditErr)
	assert.Empty(t, storage.deletedPath)
}

func TestUseCase_Delete_AuditsBeforeStorageDelete(t *testing.T) {
	t.Parallel()

	storageErr := errors.New("storage unavailable")
	storage := &fakeStorageAdminStorage{deleteErr: storageErr}
	auditRepo := &fakeStorageAdminAuditLogRepo{}
	uc := NewUseCase(Deps{Storage: storage, AuditLog: auditRepo})

	err := uc.Delete(context.Background(), usecase.StorageAdminDeleteParams{
		Path:    "challenges/file.zip",
		ActorID: uuid.New(),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, storageErr)
	assert.Equal(t, "challenges/file.zip", storage.deletedPath)
	require.NotNil(t, auditRepo.log)
	assert.Equal(t, "challenges/file.zip", auditRepo.log.Details["path"])
}

func TestUseCase_Delete_RequiresActor(t *testing.T) {
	t.Parallel()

	storage := &fakeStorageAdminStorage{}
	uc := NewUseCase(Deps{Storage: storage, AuditLog: &fakeStorageAdminAuditLogRepo{}})

	err := uc.Delete(context.Background(), usecase.StorageAdminDeleteParams{Path: "challenges/file.zip"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "actor_id is required")
	assert.Empty(t, storage.deletedPath)
}

func TestUseCase_List_RequiresPrefixAndAppliesDefaultLimit(t *testing.T) {
	t.Parallel()

	storage := &fakeStorageAdminStorage{}
	uc := NewUseCase(Deps{Storage: storage, AuditLog: &fakeStorageAdminAuditLogRepo{}})

	paths, err := uc.List(context.Background(), usecase.StorageAdminListParams{Prefix: "challenges/"})

	require.NoError(t, err)
	assert.Equal(t, []string{"challenges/file.zip"}, paths)
	assert.Equal(t, "challenges/", storage.listPrefix)
	assert.Equal(t, defaultStorageListLimit, storage.listLimit)

	_, err = uc.List(context.Background(), usecase.StorageAdminListParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prefix is required")
}

func TestUseCase_List_RejectsLimitAboveMax(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(Deps{Storage: &fakeStorageAdminStorage{}, AuditLog: &fakeStorageAdminAuditLogRepo{}})

	_, err := uc.List(context.Background(), usecase.StorageAdminListParams{
		Prefix: "challenges/",
		Limit:  maxStorageListLimit + 1,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be between")
}
