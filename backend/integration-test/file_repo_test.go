package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create Tests

func TestFileRepo_Create(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "file_upload", 100)

	file := &entity.File{
		Type:        entity.FileTypeChallenge,
		ChallengeId: challenge.Id,
		Location:    "/tmp/test_file.txt",
		Filename:    "test_file.txt",
		Size:        1024,
		SHA256:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		CreatedAt:   time.Now().UTC(),
	}

	err := f.FileRepo.Create(ctx, file)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, file.Id)
}

func TestFileRepo_Create_InvalidChallengeID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	file := &entity.File{
		Type:        entity.FileTypeChallenge,
		ChallengeId: uuid.New(),
		Location:    "/tmp/fail.txt",
		Filename:    "fail.txt",
		Size:        123,
		SHA256:      "hash",
	}

	err := f.FileRepo.Create(ctx, file)
	assert.Error(t, err)
}

// GetByID Tests

func TestFileRepo_GetByID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "get_by_id", 100)

	file := &entity.File{
		Type:        entity.FileTypeChallenge,
		ChallengeId: challenge.Id,
		Location:    "loc",
		Filename:    "name",
		Size:        100,
		SHA256:      "hash",
		CreatedAt:   time.Now().UTC(),
	}
	err := f.FileRepo.Create(ctx, file)
	require.NoError(t, err)

	got, err := f.FileRepo.GetByID(ctx, file.Id)
	assert.NoError(t, err)
	assert.Equal(t, file.Id, got.Id)
	assert.Equal(t, file.Filename, got.Filename)
}

func TestFileRepo_GetByID_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	got, err := f.FileRepo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, entityError.ErrFileNotFound)
	assert.Nil(t, got)
}

// GetByChallengeID Tests

func TestFileRepo_GetByChallengeID(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "list_files", 100)

	file1 := &entity.File{Type: entity.FileTypeChallenge, ChallengeId: challenge.Id, Location: "1", Filename: "1", Size: 1, SHA256: "1"}
	file2 := &entity.File{Type: entity.FileTypeChallenge, ChallengeId: challenge.Id, Location: "2", Filename: "2", Size: 2, SHA256: "2"}

	require.NoError(t, f.FileRepo.Create(ctx, file1))
	require.NoError(t, f.FileRepo.Create(ctx, file2))

	files, err := f.FileRepo.GetByChallengeID(ctx, challenge.Id, entity.FileTypeChallenge)
	assert.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestFileRepo_GetByChallengeID_Empty(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "empty_files", 100)

	files, err := f.FileRepo.GetByChallengeID(ctx, challenge.Id, entity.FileTypeChallenge)
	assert.NoError(t, err)
	assert.Empty(t, files)
}

// Delete Tests

func TestFileRepo_Delete(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "del_file", 100)
	file := &entity.File{Type: entity.FileTypeChallenge, ChallengeId: challenge.Id, Location: "d", Filename: "d", Size: 1, SHA256: "d"}
	require.NoError(t, f.FileRepo.Create(ctx, file))

	err := f.FileRepo.Delete(ctx, file.Id)
	assert.NoError(t, err)

	_, err = f.FileRepo.GetByID(ctx, file.Id)
	assert.ErrorIs(t, err, entityError.ErrFileNotFound)
}

func TestFileRepo_Delete_NotFound(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.FileRepo.Delete(ctx, uuid.New())
	assert.ErrorIs(t, err, entityError.ErrFileNotFound)
}
