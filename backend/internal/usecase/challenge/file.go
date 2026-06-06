package challenge

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type FileUseCase struct {
	deps FileDeps
}

type PageReader interface {
	GetByID(ctx context.Context, ID uuid.UUID) (*domain.Page, error)
}

type FileDeps struct {
	FileRepo       repo.FileRepository
	ChallengeRepo  repo.ChallengeRepository
	PageRepo       PageReader
	SolveRepo      repo.SolveRepository
	Storage        FileStorage
	Expiry         time.Duration
	DownloadSecret string
	BaseURL        string
}

var _ usecase.FileUseCase = (*FileUseCase)(nil)

func NewFileUseCase(deps FileDeps) *FileUseCase {
	return &FileUseCase{deps: deps}
}

func (uc *FileUseCase) GetByPageID(ctx context.Context, pageID uuid.UUID) ([]*domain.File, error) {
	files, err := uc.deps.FileRepo.GetByPageID(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - GetByPageID: %w", err)
	}

	return files, nil
}

func (uc *FileUseCase) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	rc, err := uc.deps.Storage.Download(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - Download - Storage.Download: %w", err)
	}

	return rc, nil
}

func (uc *FileUseCase) GetDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error) {
	file, err := uc.deps.FileRepo.GetByID(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("FileUseCase - GetDownloadURL - FileRepo.GetByID: %w", err)
	}

	url, err := uc.deps.Storage.GetPresignedURL(ctx, file.Location, uc.deps.Expiry)
	if err != nil {
		return "", fmt.Errorf("FileUseCase - GetDownloadURL - Storage.Presign: %w", err)
	}

	return url, nil
}

func (uc *FileUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType) ([]*domain.File, error) {
	files, err := uc.deps.FileRepo.GetByChallengeID(ctx, challengeID, fileType)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - GetByChallengeID - FileRepo.GetByChallengeID: %w", err)
	}

	return files, nil
}
