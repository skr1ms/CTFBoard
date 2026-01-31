package challenge

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFileUseCase_Upload(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		challengeID := uuid.New()
		fileType := entity.FileTypeChallenge
		filename := "test_task.txt"
		content := []byte("test content")
		reader := bytes.NewReader(content)
		size := int64(len(content))
		contentType := "text/plain"

		deps.s3Provider.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, size, contentType).Return(nil)
		deps.fileRepo.On("Create", ctx, mock.MatchedBy(func(f *entity.File) bool {
			return f.ChallengeID == challengeID &&
				f.Filename == filename &&
				f.Size == size &&
				f.Type == fileType
		})).Return(nil)

		file, err := uc.Upload(ctx, challengeID, fileType, filename, reader, size, contentType)

		assert.NoError(t, err)
		assert.NotNil(t, file)
		assert.Equal(t, filename, file.Filename)
		assert.Equal(t, challengeID, file.ChallengeID)

		deps.s3Provider.AssertExpectations(t)
		deps.fileRepo.AssertExpectations(t)
	})

	t.Run("Error_StorageUploadFails", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		challengeID := uuid.New()
		fileType := entity.FileTypeChallenge
		filename := "test_task.txt"
		reader := bytes.NewReader([]byte("test"))
		size := int64(4)
		contentType := "text/plain"

		expectedErr := errors.New("storage error")

		deps.s3Provider.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, size, contentType).Return(expectedErr)

		file, err := uc.Upload(ctx, challengeID, fileType, filename, reader, size, contentType)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "Storage")
		assert.Contains(t, err.Error(), expectedErr.Error())

		deps.s3Provider.AssertExpectations(t)
		deps.fileRepo.AssertNotCalled(t, "Create")
	})
}

func TestFileUseCase_Download(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		path := "some/path/file.txt"
		mockReadCloser := io.NopCloser(bytes.NewReader([]byte("content")))

		deps.s3Provider.On("Download", ctx, path).Return(mockReadCloser, nil)

		rc, err := uc.Download(ctx, path)
		assert.NoError(t, err)
		assert.NotNil(t, rc)

		deps.s3Provider.AssertExpectations(t)
	})

	t.Run("Error_StorageFails", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		path := "some/path/file.txt"
		expectedErr := errors.New("storage fail")

		deps.s3Provider.On("Download", ctx, path).Return(nil, expectedErr)

		rc, err := uc.Download(ctx, path)
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Equal(t, expectedErr, err)

		deps.s3Provider.AssertExpectations(t)
	})
}

func TestFileUseCase_GetDownloadURL(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		fileID := uuid.New()
		fileEntity := &entity.File{
			ID:       fileID,
			Location: "loc/path/file.txt",
		}
		expectedURL := "http://example.com/file"

		deps.fileRepo.On("GetByID", ctx, fileID).Return(fileEntity, nil)
		deps.s3Provider.On("GetPresignedURL", ctx, fileEntity.Location, time.Hour).Return(expectedURL, nil)

		url, err := uc.GetDownloadURL(ctx, fileID)
		assert.NoError(t, err)
		assert.Equal(t, expectedURL, url)

		deps.fileRepo.AssertExpectations(t)
		deps.s3Provider.AssertExpectations(t)
	})

	t.Run("Error_FileNotFound", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		fileID := uuid.New()

		deps.fileRepo.On("GetByID", ctx, fileID).Return(nil, entityError.ErrFileNotFound)

		url, err := uc.GetDownloadURL(ctx, fileID)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "GetByID")

		deps.fileRepo.AssertExpectations(t)
		deps.s3Provider.AssertNotCalled(t, "GetPresignedURL")
	})
}

func TestFileUseCase_GetByChallengeID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		challengeID := uuid.New()
		fileType := entity.FileTypeChallenge
		expectedFiles := []*entity.File{
			{ID: uuid.New(), Filename: "f1"},
			{ID: uuid.New(), Filename: "f2"},
		}

		deps.fileRepo.On("GetByChallengeID", ctx, challengeID, fileType).Return(expectedFiles, nil)

		files, err := uc.GetByChallengeID(ctx, challengeID, fileType)
		assert.NoError(t, err)
		assert.Equal(t, expectedFiles, files)

		deps.fileRepo.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		challengeID := uuid.New()
		fileType := entity.FileTypeChallenge
		expectedErr := errors.New("db error")

		deps.fileRepo.On("GetByChallengeID", ctx, challengeID, fileType).Return(nil, expectedErr)

		files, err := uc.GetByChallengeID(ctx, challengeID, fileType)
		assert.Error(t, err)
		assert.Nil(t, files)

		deps.fileRepo.AssertExpectations(t)
	})
}

func TestFileUseCase_Delete(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		fileID := uuid.New()
		fileEntity := &entity.File{ID: fileID, Location: "loc"}

		deps.fileRepo.On("GetByID", ctx, fileID).Return(fileEntity, nil)
		deps.s3Provider.On("Delete", ctx, fileEntity.Location).Return(nil)
		deps.fileRepo.On("Delete", ctx, fileID).Return(nil)

		err := uc.Delete(ctx, fileID)
		assert.NoError(t, err)

		deps.fileRepo.AssertExpectations(t)
		deps.s3Provider.AssertExpectations(t)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		fileID := uuid.New()

		deps.fileRepo.On("GetByID", ctx, fileID).Return(nil, entityError.ErrFileNotFound)

		err := uc.Delete(ctx, fileID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GetByID")

		deps.fileRepo.AssertExpectations(t)
		deps.s3Provider.AssertNotCalled(t, "Delete")
	})

	t.Run("Error_StorageDeleteFails", func(t *testing.T) {
		h := NewChallengeTestHelper(t)
		deps := h.Deps()
		uc := h.CreateFileUseCase()

		ctx := context.Background()
		fileID := uuid.New()
		fileEntity := &entity.File{ID: fileID, Location: "loc"}
		expectedErr := errors.New("s3 err")

		deps.fileRepo.On("GetByID", ctx, fileID).Return(fileEntity, nil)
		deps.s3Provider.On("Delete", ctx, fileEntity.Location).Return(expectedErr)

		err := uc.Delete(ctx, fileID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Storage")

		deps.fileRepo.AssertExpectations(t)
		deps.s3Provider.AssertExpectations(t)
	})
}
