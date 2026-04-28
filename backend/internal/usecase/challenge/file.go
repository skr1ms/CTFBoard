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

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

type FileUseCase struct {
	deps FileDeps
}

type FileDeps struct {
	FileRepo       repo.FileRepository
	ChallengeRepo  repo.ChallengeRepository
	PageRepo       repo.PageRepository
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

	storagePath, err = storage.GenerateStoragePath(filename)
	if err != nil {
		return "", "", fmt.Errorf("FileUseCase - uploadToStorage - GenerateStoragePath: %w", err)
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

func (uc *FileUseCase) VerifyDownloadTokenAndGetFile(ctx context.Context, path, token string, teamID *uuid.UUID) (*domain.File, error) {
	fileID, err := uc.VerifyDownloadToken(token, teamID)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - VerifyDownloadTokenAndGetFile - JWT.Validate: %w", err)
	}

	file, err := uc.deps.FileRepo.GetByLocation(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - VerifyDownloadTokenAndGetFile - FileRepo.GetByLocation: %w", err)
	}

	if file.ID != fileID {
		return nil, apperr.ErrFileIDMismatch
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

// GetByChallengeIDWithAccess returns files for a challenge filtered by fileType, enforcing
// access control for non-admin callers. It checks that the challenge is not hidden and
// that all prerequisite challenges are satisfied by the team. For writeup files it
// additionally requires that the team has an existing solve record for the challenge
// missing either condition returns ErrWriteupAccessDenied. Admin callers bypass all checks.
func (uc *FileUseCase) GetByChallengeIDWithAccess(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType, teamID *uuid.UUID, isAdmin bool) ([]*domain.File, error) {
	if !isAdmin {
		challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
		if err != nil {
			return nil, fmt.Errorf("FileUseCase - GetByChallengeIDWithAccess - ChallengeRepo.GetByID: %w", err)
		}

		if err := guard.EnsureChallengeVisible(challenge); err != nil {
			return nil, err
		}

		reqs, err := uc.deps.ChallengeRepo.GetRequirements(ctx, challengeID)
		if err != nil {
			return nil, fmt.Errorf("FileUseCase - GetByChallengeIDWithAccess - GetRequirements: %w", err)
		}

		if len(reqs) > 0 {
			if teamID == nil || uc.deps.SolveRepo == nil {
				return nil, apperr.ErrChallengeNotFound
			}

			met, err := requirementsMet(ctx, challengeID, *teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
			if err != nil {
				return nil, fmt.Errorf("FileUseCase - GetByChallengeIDWithAccess - requirementsMet: %w", err)
			}

			if !met {
				return nil, apperr.ErrChallengeNotFound
			}
		}
	}

	if fileType == domain.FileTypeWriteup && !isAdmin {
		if teamID == nil {
			return nil, apperr.ErrWriteupAccessDenied
		}

		if _, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, *teamID, challengeID); err != nil {
			if errors.Is(err, apperr.ErrSolveNotFound) {
				return nil, apperr.ErrWriteupAccessDenied
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

// GenerateDownloadToken produces a base64-encoded HMAC-SHA256 signed download token
// encoding fileID, optional teamID, and expiry Unix timestamp.
func (uc *FileUseCase) GenerateDownloadToken(fileID uuid.UUID, teamID *uuid.UUID, expiry time.Time) string {
	expirySec := expiry.Unix()
	teamStr := ""

	if teamID != nil {
		teamStr = teamID.String()
	}

	message := fmt.Sprintf("%s:%s:%d", fileID.String(), teamStr, expirySec)
	signature := crypto.HMACSign([]byte(uc.deps.DownloadSecret), []byte(message))
	token := fmt.Sprintf("%s:%s:%d:%s", fileID.String(), teamStr, expirySec, base64.URLEncoding.EncodeToString(signature))

	return base64.URLEncoding.EncodeToString([]byte(token))
}

// VerifyDownloadToken decodes a base64url download token, parses its four colon-separated
// fields (fileID, teamID string, expiry unix timestamp, base64url HMAC signature), and
// validates them in order: format, expiry, team binding (the token's team field must match
// the caller's teamID - both empty string for anonymous tokens), and finally HMAC
// authenticity. Returns the fileID on success, or one of ErrFileInvalidToken /
// ErrFileTokenExpired on failure.
func (uc *FileUseCase) VerifyDownloadToken(token string, teamID *uuid.UUID) (uuid.UUID, error) {
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	parts := strings.Split(string(tokenBytes), ":")
	if len(parts) != 4 {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	fileID, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	tokenTeamStr := parts[1]

	expirySec, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	if time.Now().Unix() > expirySec {
		return uuid.Nil, apperr.ErrFileTokenExpired
	}

	callerTeamStr := ""

	if teamID != nil {
		callerTeamStr = teamID.String()
	}

	if tokenTeamStr != callerTeamStr {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	signature, err := base64.URLEncoding.DecodeString(parts[3])
	if err != nil {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	message := fmt.Sprintf("%s:%s:%d", fileID.String(), tokenTeamStr, expirySec)
	if !crypto.HMACVerify([]byte(uc.deps.DownloadSecret), []byte(message), signature) {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	return fileID, nil
}

// GetDownloadURLWithAccess verifies that the caller may access the requested file, then
// returns an HMAC-signed short-lived download token embedded in a full download URL
// For non-admin callers it enforces: challenge visibility (not hidden), prerequisite
// satisfaction, and - for writeup files - that the team has a solve record. The token is
// generated by GenerateDownloadToken with an expiry of FileDeps.Expiry from now and bound
// to the caller's teamID so it cannot be reused by a different team.
func (uc *FileUseCase) GetDownloadURLWithAccess(ctx context.Context, fileID uuid.UUID, teamID *uuid.UUID, isAdmin bool) (string, error) {
	file, err := uc.deps.FileRepo.GetByID(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("FileUseCase - GetDownloadURLWithAccess - FileRepo.GetByID: %w", err)
	}

	if !isAdmin && file.ChallengeID != nil {
		challenge, errChal := uc.deps.ChallengeRepo.GetByID(ctx, *file.ChallengeID)
		if errChal != nil {
			return "", fmt.Errorf("FileUseCase - GetDownloadURLWithAccess - ChallengeRepo.GetByID: %w", errChal)
		}

		if err := guard.EnsureChallengeVisible(challenge); err != nil {
			return "", err
		}

		reqs, err := uc.deps.ChallengeRepo.GetRequirements(ctx, *file.ChallengeID)
		if err != nil {
			return "", fmt.Errorf("FileUseCase - GetDownloadURLWithAccess - GetRequirements: %w", err)
		}

		if len(reqs) > 0 {
			if teamID == nil || uc.deps.SolveRepo == nil {
				return "", apperr.ErrChallengeNotFound
			}

			met, err := requirementsMet(ctx, *file.ChallengeID, *teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
			if err != nil {
				return "", fmt.Errorf("FileUseCase - GetDownloadURLWithAccess - requirementsMet: %w", err)
			}

			if !met {
				return "", apperr.ErrChallengeNotFound
			}
		}
	}

	if file.Type == domain.FileTypeWriteup && !isAdmin && file.ChallengeID != nil {
		if teamID == nil {
			return "", apperr.ErrWriteupAccessDenied
		}

		_, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, *teamID, *file.ChallengeID)
		if err != nil {
			if errors.Is(err, apperr.ErrSolveNotFound) {
				return "", apperr.ErrWriteupAccessDenied
			}

			return "", fmt.Errorf("FileUseCase - GetDownloadURLWithAccess - SolveRepo.GetByTeamAndChallenge: %w", err)
		}
	}

	expiry := time.Now().Add(uc.deps.Expiry)
	token := uc.GenerateDownloadToken(fileID, teamID, expiry)
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
	// to a missing object if the DB delete fails after storage is already gone
	if err := uc.deps.FileRepo.Delete(ctx, fileID); err != nil {
		return fmt.Errorf("FileUseCase - Delete - FileRepo.Delete: %w", err)
	}

	if err := uc.deps.Storage.Delete(ctx, file.Location); err != nil {
		return fmt.Errorf("FileUseCase - Delete - Storage.Delete: %w", err)
	}

	return nil
}
