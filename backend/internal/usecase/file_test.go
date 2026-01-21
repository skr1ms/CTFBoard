package usecase_test

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
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFileUseCase_Upload(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		challengeID := uuid.New()
		fileType := entity.FileTypeChallenge
		filename := "test_task.txt"
		content := []byte("test content")
		reader := bytes.NewReader(content)
		size := int64(len(content))
		contentType := "text/plain"

		mockStorage.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, size, contentType).Return(nil)

		mockRepo.On("Create", ctx, mock.MatchedBy(func(f *entity.File) bool {
			return f.ChallengeId == challengeID &&
				f.Filename == filename &&
				f.Size == size &&
				f.Type == fileType
		})).Return(nil)

		file, err := uc.Upload(ctx, challengeID, fileType, filename, reader, size, contentType)

		assert.NoError(t, err)
		assert.NotNil(t, file)
		assert.Equal(t, filename, file.Filename)
		assert.Equal(t, challengeID, file.ChallengeId)

		mockStorage.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error_StorageUploadFails", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		challengeID := uuid.New()
		fileType := entity.FileTypeChallenge
		filename := "test_task.txt"
		reader := bytes.NewReader([]byte("test"))
		size := int64(4)
		contentType := "text/plain"

		expectedErr := errors.New("storage error")

		mockStorage.On("Upload", ctx, mock.AnythingOfType("string"), mock.Anything, size, contentType).Return(expectedErr)

		file, err := uc.Upload(ctx, challengeID, fileType, filename, reader, size, contentType)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "Storage")
		assert.Contains(t, err.Error(), expectedErr.Error())

		mockStorage.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Create")
	})
}

func TestFileUseCase_Download(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		path := "some/path/file.txt"
		mockReadCloser := io.NopCloser(bytes.NewReader([]byte("content")))

		mockStorage.On("Download", ctx, path).Return(mockReadCloser, nil)

		rc, err := uc.Download(ctx, path)
		assert.NoError(t, err)
		assert.NotNil(t, rc)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Error_StorageFails", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		path := "some/path/file.txt"
		expectedErr := errors.New("storage fail")

		mockStorage.On("Download", ctx, path).Return(nil, expectedErr)

		rc, err := uc.Download(ctx, path)
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Equal(t, expectedErr, err)

		mockStorage.AssertExpectations(t)
	})
}

func TestFileUseCase_GetDownloadURL(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		fileId := uuid.New()
		fileEntity := &entity.File{
			Id:       fileId,
			Location: "loc/path/file.txt",
		}
		expectedUrl := "http://example.com/file"

		mockRepo.On("GetByID", ctx, fileId).Return(fileEntity, nil)
		mockStorage.On("GetPresignedURL", ctx, fileEntity.Location, time.Hour).Return(expectedUrl, nil)

		url, err := uc.GetDownloadURL(ctx, fileId)
		assert.NoError(t, err)
		assert.Equal(t, expectedUrl, url)

		mockRepo.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Error_FileNotFound", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		fileId := uuid.New()

		mockRepo.On("GetByID", ctx, fileId).Return(nil, entityError.ErrFileNotFound)

		url, err := uc.GetDownloadURL(ctx, fileId)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "GetByID")

		mockRepo.AssertExpectations(t)
		mockStorage.AssertNotCalled(t, "GetPresignedURL")
	})
}

func TestFileUseCase_GetByChallengeID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		challengeId := uuid.New()
		fileType := entity.FileTypeChallenge
		expectedFiles := []*entity.File{
			{Id: uuid.New(), Filename: "f1"},
			{Id: uuid.New(), Filename: "f2"},
		}

		mockRepo.On("GetByChallengeID", ctx, challengeId, fileType).Return(expectedFiles, nil)

		files, err := uc.GetByChallengeID(ctx, challengeId, fileType)
		assert.NoError(t, err)
		assert.Equal(t, expectedFiles, files)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		challengeId := uuid.New()
		fileType := entity.FileTypeChallenge
		expectedErr := errors.New("db error")

		mockRepo.On("GetByChallengeID", ctx, challengeId, fileType).Return(nil, expectedErr)

		files, err := uc.GetByChallengeID(ctx, challengeId, fileType)
		assert.Error(t, err)
		assert.Nil(t, files)

		mockRepo.AssertExpectations(t)
	})
}

func TestFileUseCase_Delete(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		fileId := uuid.New()
		fileEntity := &entity.File{Id: fileId, Location: "loc"}

		mockRepo.On("GetByID", ctx, fileId).Return(fileEntity, nil)
		mockStorage.On("Delete", ctx, fileEntity.Location).Return(nil)
		mockRepo.On("Delete", ctx, fileId).Return(nil)

		err := uc.Delete(ctx, fileId)
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		fileId := uuid.New()

		mockRepo.On("GetByID", ctx, fileId).Return(nil, entityError.ErrFileNotFound)

		err := uc.Delete(ctx, fileId)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GetByID")

		mockRepo.AssertExpectations(t)
		mockStorage.AssertNotCalled(t, "Delete")
	})

	t.Run("Error_StorageDeleteFails", func(t *testing.T) {
		mockRepo := mocks.NewMockFileRepository(t)
		mockStorage := mocks.NewMockS3Provider(t)
		uc := usecase.NewFileUseCase(mockRepo, mockStorage, time.Hour)

		ctx := context.Background()
		fileId := uuid.New()
		fileEntity := &entity.File{Id: fileId, Location: "loc"}
		expectedErr := errors.New("s3 err")

		mockRepo.On("GetByID", ctx, fileId).Return(fileEntity, nil)
		mockStorage.On("Delete", ctx, fileEntity.Location).Return(expectedErr)

		err := uc.Delete(ctx, fileId)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Storage")

		mockRepo.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
	})
}
