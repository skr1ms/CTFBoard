package challenge

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type FileUseCase struct {
	deps FileDeps
}

type FileDeps struct {
	FileRepo       repo.FileRepository
	ChallengeRepo  repo.ChallengeRepository
	SolveRepo      repo.SolveRepository
	Storage        storage.Provider
	Expiry         time.Duration
	DownloadSecret string
	BaseURL        string
}

var _ usecase.FileUseCase = (*FileUseCase)(nil)

func NewFileUseCase(deps FileDeps) *FileUseCase {
	return &FileUseCase{deps: deps}
}

func (uc *FileUseCase) Upload(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType, filename string, reader io.Reader, size int64, contentType string) (*domain.File, error) {
	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
		return nil, err
	}

	tempFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - Upload - os.CreateTemp: %w", err)
	}

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	hash := sha256.New()
	multiWriter := io.MultiWriter(tempFile, hash)

	if _, err := io.Copy(multiWriter, reader); err != nil {
		return nil, fmt.Errorf("FileUseCase - Upload - io.Copy: %w", err)
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("FileUseCase - Upload - file.Seek: %w", err)
	}

	sha256Hash := crypto.HashHex(hash)

	storagePath, err := storage.GenerateStoragePath(filename)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - Upload - GenerateStoragePath: %w", err)
	}

	if err := uc.deps.Storage.Upload(ctx, storagePath, tempFile, size, contentType); err != nil {
		return nil, fmt.Errorf("FileUseCase - Upload - Storage.Put: %w", err)
	}

	file := &domain.File{
		Type:        fileType,
		ChallengeID: challengeID,
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

func (uc *FileUseCase) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	rc, err := uc.deps.Storage.Download(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - Download - Storage.Download: %w", err)
	}

	return rc, nil
}

func (uc *FileUseCase) VerifyDownloadTokenAndGetFile(ctx context.Context, path, token string) (*domain.File, error) {
	fileID, err := uc.VerifyDownloadToken(token)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - VerifyDownloadTokenAndGetFile - JWT.Validate: %w", err)
	}

	file, err := uc.deps.FileRepo.GetByLocation(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - VerifyDownloadTokenAndGetFile - FileRepo.GetByLocation: %w", err)
	}

	if file.ID != fileID {
		return nil, httperr.ErrFileIDMismatch
	}

	return file, nil
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

// GetByChallengeIDWithAccess: for writeup type, returns only if teamID solved the challenge or caller is admin.
// Returns ErrChallengeNotFound if the challenge is hidden and caller is not admin.
func (uc *FileUseCase) GetByChallengeIDWithAccess(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType, teamID *uuid.UUID, isAdmin bool) ([]*domain.File, error) {
	if !isAdmin {
		challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
		if err != nil {
			return nil, fmt.Errorf("FileUseCase - GetByChallengeIDWithAccess - ChallengeRepo.GetByID: %w", err)
		}

		if challenge.State == domain.ChallengeStateHidden {
			return nil, httperr.ErrChallengeNotFound
		}

		reqs, err := uc.deps.ChallengeRepo.GetRequirements(ctx, challengeID)
		if err != nil {
			return nil, fmt.Errorf("FileUseCase - GetByChallengeIDWithAccess - GetRequirements: %w", err)
		}

		if len(reqs) > 0 {
			if teamID == nil || uc.deps.SolveRepo == nil {
				return nil, httperr.ErrChallengeNotFound
			}

			met, err := requirementsMet(ctx, challengeID, *teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
			if err != nil {
				return nil, fmt.Errorf("FileUseCase - GetByChallengeIDWithAccess - requirementsMet: %w", err)
			}

			if !met {
				return nil, httperr.ErrChallengeNotFound
			}
		}
	}

	if fileType == domain.FileTypeWriteup && !isAdmin {
		if teamID == nil {
			return nil, httperr.ErrWriteupAccessDenied
		}

		if _, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, *teamID, challengeID); err != nil {
			if errors.Is(err, httperr.ErrSolveNotFound) {
				return nil, httperr.ErrWriteupAccessDenied
			}

			return nil, fmt.Errorf("FileUseCase - GetByChallengeIDWithAccess - SolveRepo.GetByTeamAndChallenge: %w", err)
		}
	}

	files, err := uc.deps.FileRepo.GetByChallengeID(ctx, challengeID, fileType)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - GetByChallengeIDWithAccess - FileRepo.GetByChallengeID: %w", err)
	}

	return files, nil
}

func (uc *FileUseCase) GenerateDownloadToken(fileID uuid.UUID, expiry time.Time) string {
	expiryUnix := expiry.Unix()
	message := fmt.Sprintf("%s:%d", fileID.String(), expiryUnix)
	signature := crypto.HMACSign([]byte(uc.deps.DownloadSecret), []byte(message))
	token := fmt.Sprintf("%s:%d:%s", fileID.String(), expiryUnix, base64.URLEncoding.EncodeToString(signature))

	return base64.URLEncoding.EncodeToString([]byte(token))
}

func (uc *FileUseCase) VerifyDownloadToken(token string) (uuid.UUID, error) {
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return uuid.Nil, httperr.ErrFileInvalidToken
	}

	parts := strings.Split(string(tokenBytes), ":")
	if len(parts) != 3 {
		return uuid.Nil, httperr.ErrFileInvalidToken
	}

	fileID, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, httperr.ErrFileInvalidToken
	}

	expiryUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return uuid.Nil, httperr.ErrFileInvalidToken
	}

	if time.Now().Unix() > expiryUnix {
		return uuid.Nil, httperr.ErrFileTokenExpired
	}

	signature, err := base64.URLEncoding.DecodeString(parts[2])
	if err != nil {
		return uuid.Nil, httperr.ErrFileInvalidToken
	}

	message := fmt.Sprintf("%s:%d", fileID.String(), expiryUnix)
	if !crypto.HMACVerify([]byte(uc.deps.DownloadSecret), []byte(message), signature) {
		return uuid.Nil, httperr.ErrFileInvalidToken
	}

	return fileID, nil
}

// GetDownloadURLWithAccess: for writeup type, returns error unless teamID solved the challenge or caller is admin.
// Returns ErrChallengeNotFound if the challenge is hidden and caller is not admin.
func (uc *FileUseCase) GetDownloadURLWithAccess(ctx context.Context, fileID uuid.UUID, teamID *uuid.UUID, isAdmin bool) (string, error) {
	file, err := uc.deps.FileRepo.GetByID(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("FileUseCase - GetDownloadURLWithAccess - FileRepo.GetByID: %w", err)
	}

	if !isAdmin {
		challenge, errChal := uc.deps.ChallengeRepo.GetByID(ctx, file.ChallengeID)
		if errChal != nil {
			return "", fmt.Errorf("FileUseCase - GetDownloadURLWithAccess - ChallengeRepo.GetByID: %w", errChal)
		}

		if challenge.State == domain.ChallengeStateHidden {
			return "", httperr.ErrChallengeNotFound
		}

		reqs, err := uc.deps.ChallengeRepo.GetRequirements(ctx, file.ChallengeID)
		if err != nil {
			return "", fmt.Errorf("FileUseCase - GetDownloadURLWithAccess - GetRequirements: %w", err)
		}

		if len(reqs) > 0 {
			if teamID == nil || uc.deps.SolveRepo == nil {
				return "", httperr.ErrChallengeNotFound
			}

			met, err := requirementsMet(ctx, file.ChallengeID, *teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
			if err != nil {
				return "", fmt.Errorf("FileUseCase - GetDownloadURLWithAccess - requirementsMet: %w", err)
			}

			if !met {
				return "", httperr.ErrChallengeNotFound
			}
		}
	}

	if file.Type == domain.FileTypeWriteup && !isAdmin {
		if teamID == nil {
			return "", httperr.ErrWriteupAccessDenied
		}

		_, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, *teamID, file.ChallengeID)
		if err != nil {
			if errors.Is(err, httperr.ErrSolveNotFound) {
				return "", httperr.ErrWriteupAccessDenied
			}

			return "", fmt.Errorf("FileUseCase - GetDownloadURLWithAccess - SolveRepo.GetByTeamAndChallenge: %w", err)
		}
	}

	expiry := time.Now().Add(uc.deps.Expiry)
	token := uc.GenerateDownloadToken(fileID, expiry)
	downloadURL := fmt.Sprintf("%s/api/v1/files/download/%s?token=%s", uc.deps.BaseURL, escapeLocationPath(file.Location), url.QueryEscape(token))

	return downloadURL, nil
}

func escapeLocationPath(location string) string {
	if location == "" {
		return ""
	}

	parts := strings.Split(location, "/")

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}

		out = append(out, url.PathEscape(p))
	}

	return strings.Join(out, "/")
}

func (uc *FileUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType) ([]*domain.File, error) {
	files, err := uc.deps.FileRepo.GetByChallengeID(ctx, challengeID, fileType)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - GetByChallengeID - FileRepo.GetByChallengeID: %w", err)
	}

	return files, nil
}

func (uc *FileUseCase) Delete(ctx context.Context, fileID uuid.UUID) error {
	file, err := uc.deps.FileRepo.GetByID(ctx, fileID)
	if err != nil {
		return fmt.Errorf("FileUseCase - Delete - FileRepo.GetByID: %w", err)
	}
	// Delete the DB record first. If it succeeds but the storage delete below fails,
	// the orphaned object in storage can be garbage-collected later without the app
	// serving a broken reference. The reverse order risks a broken DB record pointing
	// to a missing object if the DB delete fails after storage is already gone.
	if err := uc.deps.FileRepo.Delete(ctx, fileID); err != nil {
		return fmt.Errorf("FileUseCase - Delete - FileRepo.Delete: %w", err)
	}

	if err := uc.deps.Storage.Delete(ctx, file.Location); err != nil {
		return fmt.Errorf("FileUseCase - Delete - Storage.Delete: %w", err)
	}

	return nil
}
