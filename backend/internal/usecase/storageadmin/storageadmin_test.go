package storageadmin

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

type storageAdminFakeStorage struct {
	listPrefix string
	listPaths  []string
	listErr    error
	listCalls  int

	deletePath  string
	deleteErr   error
	deleteCalls int
}

func (s *storageAdminFakeStorage) List(_ context.Context, prefix string) ([]string, error) {
	s.listPrefix = prefix
	s.listCalls++

	return s.listPaths, s.listErr
}

func (s *storageAdminFakeStorage) Delete(_ context.Context, path string) error {
	s.deletePath = path
	s.deleteCalls++

	return s.deleteErr
}

func TestUseCaseListAllowsEmptyPrefix(t *testing.T) {
	t.Parallel()

	storage := &storageAdminFakeStorage{listPaths: []string{"a.txt", "dir/b.txt"}}
	uc := NewUseCase(Deps{Storage: storage})

	got, err := uc.List(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, []string{"a.txt", "dir/b.txt"}, got)
	assert.Equal(t, "", storage.listPrefix)
	assert.Equal(t, 1, storage.listCalls)
}

func TestUseCaseListRejectsUnsafePrefixes(t *testing.T) {
	t.Parallel()

	tests := []string{"../secret", "safe/../secret", "/absolute"}

	for _, prefix := range tests {
		t.Run(prefix, func(t *testing.T) {
			t.Parallel()

			storage := &storageAdminFakeStorage{}
			uc := NewUseCase(Deps{Storage: storage})

			got, err := uc.List(context.Background(), prefix)

			require.Error(t, err)
			assert.Nil(t, got)
			assert.Equal(t, 0, storage.listCalls)

			var validationErr *apperr.ValidationError
			assert.ErrorAs(t, err, &validationErr)
		})
	}
}

func TestUseCaseListWrapsStorageError(t *testing.T) {
	t.Parallel()

	storageErr := errors.New("storage unavailable")
	storage := &storageAdminFakeStorage{listErr: storageErr}
	uc := NewUseCase(Deps{Storage: storage})

	got, err := uc.List(context.Background(), "uploads/")

	require.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, storageErr)
	assert.Equal(t, "uploads/", storage.listPrefix)
	assert.Equal(t, 1, storage.listCalls)
}

func TestUseCaseDeletePassesSafePath(t *testing.T) {
	t.Parallel()

	storage := &storageAdminFakeStorage{}
	uc := NewUseCase(Deps{Storage: storage})

	err := uc.Delete(context.Background(), "uploads/file.txt")

	require.NoError(t, err)
	assert.Equal(t, "uploads/file.txt", storage.deletePath)
	assert.Equal(t, 1, storage.deleteCalls)
}

func TestUseCaseDeleteRejectsUnsafePaths(t *testing.T) {
	t.Parallel()

	tests := []string{"", "../secret", "safe/../secret", "/absolute"}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			storage := &storageAdminFakeStorage{}
			uc := NewUseCase(Deps{Storage: storage})

			err := uc.Delete(context.Background(), path)

			require.Error(t, err)
			assert.Equal(t, 0, storage.deleteCalls)

			var validationErr *apperr.ValidationError
			assert.ErrorAs(t, err, &validationErr)
		})
	}
}

func TestUseCaseDeleteWrapsStorageError(t *testing.T) {
	t.Parallel()

	storageErr := errors.New("delete failed")
	storage := &storageAdminFakeStorage{deleteErr: storageErr}
	uc := NewUseCase(Deps{Storage: storage})

	err := uc.Delete(context.Background(), "uploads/file.txt")

	require.Error(t, err)
	assert.ErrorIs(t, err, storageErr)
	assert.Equal(t, "uploads/file.txt", storage.deletePath)
	assert.Equal(t, 1, storage.deleteCalls)
}
