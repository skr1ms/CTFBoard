package challenge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/storage"
)

type FileUseCase struct {
	fileRepo repo.FileRepository
	storage  storage.Provider
	expiry   time.Duration
}

func NewFileUseCase(fileRepo repo.FileRepository, storageProvider storage.Provider, presignedExpiry time.Duration) *FileUseCase {
	return &FileUseCase{
		fileRepo: fileRepo,
		storage:  storageProvider,
		expiry:   presignedExpiry,
	}
}

func (uc *FileUseCase) Upload(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType, filename string, reader io.Reader, size int64, contentType string) (*entity.File, error) {
	tempFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - Upload - CreateTemp: %w", err)
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	hash := sha256.New()
	multiWriter := io.MultiWriter(tempFile, hash)

	if _, err := io.Copy(multiWriter, reader); err != nil {
		return nil, fmt.Errorf("FileUseCase - Upload - Copy: %w", err)
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("FileUseCase - Upload - Seek: %w", err)
	}

	sha256Hash := hex.EncodeToString(hash.Sum(nil))
	storagePath := storage.GenerateStoragePath(filename)

	if err := uc.storage.Upload(ctx, storagePath, tempFile, size, contentType); err != nil {
		return nil, fmt.Errorf("FileUseCase - Upload - Storage: %w", err)
	}

	file := &entity.File{
		Type:        fileType,
		ChallengeID: challengeID,
		Location:    storagePath,
		Filename:    filename,
		Size:        size,
		SHA256:      sha256Hash,
		CreatedAt:   time.Now(),
	}

	if err := uc.fileRepo.Create(ctx, file); err != nil {
		if delErr := uc.storage.Delete(ctx, storagePath); delErr != nil {
			return nil, fmt.Errorf("FileUseCase - Upload - Create: %w, cleanup delete: %w", err, delErr)
		}
		return nil, fmt.Errorf("FileUseCase - Upload - Create: %w", err)
	}

	return file, nil
}

func (uc *FileUseCase) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return uc.storage.Download(ctx, path)
}

func (uc *FileUseCase) GetDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error) {
	file, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("FileUseCase - GetDownloadURL - GetByID: %w", err)
	}

	url, err := uc.storage.GetPresignedURL(ctx, file.Location, uc.expiry)
	if err != nil {
		return "", fmt.Errorf("FileUseCase - GetDownloadURL - Presign: %w", err)
	}

	return url, nil
}

func (uc *FileUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType) ([]*entity.File, error) {
	files, err := uc.fileRepo.GetByChallengeID(ctx, challengeID, fileType)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - GetByChallengeID: %w", err)
	}
	return files, nil
}

func (uc *FileUseCase) Delete(ctx context.Context, fileID uuid.UUID) error {
	file, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return fmt.Errorf("FileUseCase - Delete - GetByID: %w", err)
	}

	if err := uc.storage.Delete(ctx, file.Location); err != nil {
		return fmt.Errorf("FileUseCase - Delete - Storage: %w", err)
	}

	if err := uc.fileRepo.Delete(ctx, fileID); err != nil {
		return fmt.Errorf("FileUseCase - Delete - Repo: %w", err)
	}

	return nil
}
