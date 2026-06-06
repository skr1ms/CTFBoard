package challenge

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/storagepath"
)

// uploadToStorage buffers reader into a temp file, computes its SHA-256 digest, generates
// a storage path, and uploads the file. On success it returns (storagePath, sha256Hex).
// The temporary file is deleted before the function returns regardless of outcome.
func (uc *FileUseCase) uploadToStorage(ctx context.Context, reader io.Reader, filename string, size int64, contentType string) (storagePath, sha256Hash string, err error) {
	tempFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return "", "", fmt.Errorf("FileUseCase - uploadToStorage - os.CreateTemp: %w", err)
	}

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	hash := sha256.New()
	if _, err = io.Copy(io.MultiWriter(tempFile, hash), reader); err != nil {
		return "", "", fmt.Errorf("FileUseCase - uploadToStorage - io.Copy: %w", err)
	}

	if _, err = tempFile.Seek(0, 0); err != nil {
		return "", "", fmt.Errorf("FileUseCase - uploadToStorage - file.Seek: %w", err)
	}

	storagePath, err = storagepath.Generate(filename)
	if err != nil {
		return "", "", fmt.Errorf("FileUseCase - uploadToStorage - storagepath.Generate: %w", err)
	}

	if err = uc.deps.Storage.Upload(ctx, storagePath, tempFile, size, contentType); err != nil {
		return "", "", fmt.Errorf("FileUseCase - uploadToStorage - Storage.Upload: %w", err)
	}

	return storagePath, crypto.HashHex(hash), nil
}

// Upload streams the incoming reader into a temporary file on disk while simultaneously
// computing a SHA-256 digest via io.TeeReader into an io.MultiWriter. Once streaming is
// complete the temp file is rewound and uploaded to object storage. The database record is
// inserted afterwards; if the insert fails, Upload attempts to delete the already-uploaded
// object to avoid leaving an orphan in storage, joining any deletion error with the
// original insert error before returning.
func (uc *FileUseCase) Upload(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType, filename string, reader io.Reader, size int64, contentType string) (*domain.File, error) {
	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
		return nil, err
	}

	storagePath, sha256Hash, err := uc.uploadToStorage(ctx, reader, filename, size, contentType)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - Upload - uploadToStorage: %w", err)
	}

	file := &domain.File{
		Type:        fileType,
		ChallengeID: &challengeID,
		Location:    storagePath,
		Filename:    filename,
		Size:        size,
		SHA256:      sha256Hash,
		CreatedAt:   time.Now(),
	}

	if err := uc.deps.FileRepo.Create(ctx, file); err != nil {
		delErr := uc.deps.Storage.Delete(ctx, storagePath)
		if delErr != nil {
			return nil, fmt.Errorf("FileUseCase - Upload - FileRepo.Create: %w", errors.Join(err, delErr))
		}

		return nil, fmt.Errorf("FileUseCase - Upload - FileRepo.Create: %w", err)
	}

	return file, nil
}

// UploadPageFile uploads a file attached to a static page, streaming through a temp file
// to compute SHA-256 and size before persisting to object storage and saving metadata.
func (uc *FileUseCase) UploadPageFile(ctx context.Context, pageID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (*domain.File, error) {
	if _, err := uc.deps.PageRepo.GetByID(ctx, pageID); err != nil {
		return nil, err
	}

	storagePath, sha256Hash, err := uc.uploadToStorage(ctx, reader, filename, size, contentType)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - UploadPageFile - uploadToStorage: %w", err)
	}

	file := &domain.File{
		Type:     domain.FileTypePage,
		PageID:   &pageID,
		Location: storagePath,
		Filename: filename,
		Size:     size,
		SHA256:   sha256Hash,
	}

	if err := uc.deps.FileRepo.Create(ctx, file); err != nil {
		delErr := uc.deps.Storage.Delete(ctx, storagePath)
		if delErr != nil {
			return nil, fmt.Errorf("FileUseCase - UploadPageFile - FileRepo.Create: %w", errors.Join(err, delErr))
		}

		return nil, fmt.Errorf("FileUseCase - UploadPageFile - FileRepo.Create: %w", err)
	}

	return file, nil
}

func (uc *FileUseCase) Delete(ctx context.Context, fileID uuid.UUID) error {
	file, err := uc.deps.FileRepo.GetByID(ctx, fileID)
	if err != nil {
		return fmt.Errorf("FileUseCase - Delete - FileRepo.GetByID: %w", err)
	}

	// Delete the DB record first. If storage deletion fails, the orphaned object
	// can be garbage-collected later without serving a broken DB reference.
	if err := uc.deps.FileRepo.Delete(ctx, fileID); err != nil {
		return fmt.Errorf("FileUseCase - Delete - FileRepo.Delete: %w", err)
	}

	if err := uc.deps.Storage.Delete(ctx, file.Location); err != nil {
		return fmt.Errorf("FileUseCase - Delete - Storage.Delete: %w", err)
	}

	return nil
}
