package challenge

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestFileUseCase_Upload(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		challengeID := uuid.New()
		fileType := domain.FileTypeChallenge
		filename := "test_task.txt"
		content := []byte("test content")
		reader := bytes.NewReader(content)
		size := int64(len(content))
		contentType := "text/plain"

		d.challengeRepo.On("GetByID", ctx, challengeID).Return(&domain.Challenge{ID: challengeID}, nil).Once()
		d.fileStorage.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, size, contentType).Return(nil)
		d.fileRepo.On("Create", ctx, mock.MatchedBy(func(f *domain.File) bool {
			return f.ChallengeID != nil && *f.ChallengeID == challengeID &&
				f.Filename == filename &&
				f.Size == size &&
				f.Type == fileType
		})).Return(nil)

		file, err := uc.Upload(ctx, challengeID, fileType, filename, reader, size, contentType)

		assert.NoError(t, err)
		assert.NotNil(t, file)
		assert.Equal(t, filename, file.Filename)
		assert.Equal(t, challengeID, *file.ChallengeID)

		d.fileStorage.AssertExpectations(t)
		d.fileRepo.AssertExpectations(t)
	})

	t.Run("Error_StorageUploadFails", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		challengeID := uuid.New()
		fileType := domain.FileTypeChallenge
		filename := "test_task.txt"
		reader := bytes.NewReader([]byte("test"))
		size := int64(4)
		contentType := "text/plain"

		expectedErr := errors.New("storage error")

		d.challengeRepo.On("GetByID", ctx, challengeID).Return(&domain.Challenge{ID: challengeID}, nil).Once()
		d.fileStorage.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, size, contentType).Return(expectedErr)

		file, err := uc.Upload(ctx, challengeID, fileType, filename, reader, size, contentType)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "Storage")
		assert.Contains(t, err.Error(), expectedErr.Error())

		d.fileStorage.AssertExpectations(t)
		d.fileRepo.AssertNotCalled(t, "Create")
	})
}

func TestFileUseCase_Download(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		path := "some/path/file.txt"
		mockReadCloser := io.NopCloser(bytes.NewReader([]byte("content")))

		d.fileStorage.On("Download", ctx, path).Return(mockReadCloser, nil)

		rc, err := uc.Download(ctx, path)
		assert.NoError(t, err)
		assert.NotNil(t, rc)

		d.fileStorage.AssertExpectations(t)
	})

	t.Run("Error_StorageFails", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		path := "some/path/file.txt"
		expectedErr := errors.New("storage fail")

		d.fileStorage.On("Download", ctx, path).Return(nil, expectedErr)

		rc, err := uc.Download(ctx, path)
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.ErrorIs(t, err, expectedErr, "err should wrap expectedErr")

		d.fileStorage.AssertExpectations(t)
	})
}

func TestFileUseCase_GetDownloadURL(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		fileID := uuid.New()
		fileentity := &domain.File{
			ID:       fileID,
			Location: "loc/path/file.txt",
		}
		expectedURL := "http://example.com/file"

		d.fileRepo.On("GetByID", ctx, fileID).Return(fileentity, nil)
		d.fileStorage.On("GetPresignedURL", ctx, fileentity.Location, time.Hour).Return(expectedURL, nil)

		url, err := uc.GetDownloadURL(ctx, fileID)
		assert.NoError(t, err)
		assert.Equal(t, expectedURL, url)

		d.fileRepo.AssertExpectations(t)
		d.fileStorage.AssertExpectations(t)
	})

	t.Run("Error_FileNotFound", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		fileID := uuid.New()

		d.fileRepo.On("GetByID", ctx, fileID).Return(nil, apperr.ErrFileNotFound)

		url, err := uc.GetDownloadURL(ctx, fileID)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "GetByID")

		d.fileRepo.AssertExpectations(t)
		d.fileStorage.AssertNotCalled(t, "GetPresignedURL")
	})
}

func TestFileUseCase_GetByChallengeID(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		challengeID := uuid.New()
		fileType := domain.FileTypeChallenge
		expectedFiles := []*domain.File{
			{ID: uuid.New(), Filename: "f1"},
			{ID: uuid.New(), Filename: "f2"},
		}

		d.fileRepo.On("GetByChallengeID", ctx, challengeID, fileType).Return(expectedFiles, nil)

		files, err := uc.GetByChallengeID(ctx, challengeID, fileType)
		assert.NoError(t, err)
		assert.Equal(t, expectedFiles, files)

		d.fileRepo.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		challengeID := uuid.New()
		fileType := domain.FileTypeChallenge
		expectedErr := errors.New("db error")

		d.fileRepo.On("GetByChallengeID", ctx, challengeID, fileType).Return(nil, expectedErr)

		files, err := uc.GetByChallengeID(ctx, challengeID, fileType)
		assert.Error(t, err)
		assert.Nil(t, files)

		d.fileRepo.AssertExpectations(t)
	})
}

func TestFileUseCase_Delete(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		fileID := uuid.New()
		fileentity := &domain.File{ID: fileID, Location: "loc"}

		d.fileRepo.On("GetByID", ctx, fileID).Return(fileentity, nil)
		d.fileStorage.On("Delete", ctx, fileentity.Location).Return(nil)
		d.fileRepo.On("Delete", ctx, fileID).Return(nil)

		err := uc.Delete(ctx, fileID)
		assert.NoError(t, err)

		d.fileRepo.AssertExpectations(t)
		d.fileStorage.AssertExpectations(t)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		fileID := uuid.New()

		d.fileRepo.On("GetByID", ctx, fileID).Return(nil, apperr.ErrFileNotFound)

		err := uc.Delete(ctx, fileID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GetByID")

		d.fileRepo.AssertExpectations(t)
		d.fileStorage.AssertNotCalled(t, "Delete")
	})

	t.Run("Error_StorageDeleteFails", func(t *testing.T) {
		t.Parallel()
		d := newChallengeTestDeps(t)
		uc := d.createFileUseCase()

		ctx := context.Background()
		fileID := uuid.New()
		fileentity := &domain.File{ID: fileID, Location: "loc"}
		expectedErr := errors.New("s3 err")

		d.fileRepo.On("GetByID", ctx, fileID).Return(fileentity, nil)
		d.fileRepo.On("Delete", ctx, fileID).Return(nil)
		d.fileStorage.On("Delete", ctx, fileentity.Location).Return(expectedErr)

		err := uc.Delete(ctx, fileID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Storage")

		d.fileRepo.AssertExpectations(t)
		d.fileStorage.AssertExpectations(t)
	})
}
