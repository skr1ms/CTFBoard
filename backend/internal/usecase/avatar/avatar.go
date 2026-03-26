package avatar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func NewBytesReader(data []byte) io.Reader {
	return bytes.NewReader(data)
}

type AvatarDeps struct {
	UserRepo     repo.UserRepository
	TeamRepo     repo.TeamRepository
	Storage      storage.Provider
	Cache        cachekit.KeyValueStore
	TM           repo.TransactionManager
	AuditLogRepo repo.AuditLogRepository
	Config       domain.AvatarConfig
	Logger       logkit.Logger
}

type AvatarUseCase struct {
	deps      AvatarDeps
	processor *ImageProcessor
	wg        sync.WaitGroup
}

var _ usecase.AvatarUseCase = (*AvatarUseCase)(nil)

func NewAvatarUseCase(deps AvatarDeps) *AvatarUseCase {
	return &AvatarUseCase{
		deps:      deps,
		processor: NewImageProcessor(deps.Config),
	}
}

func (uc *AvatarUseCase) Wait() { uc.wg.Wait() }

func (uc *AvatarUseCase) goDeleteOldAvatar(fullPath, thumbPath string) {
	uc.wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				uc.deps.Logger.Error("AvatarUseCase - goDeleteOldAvatar panic", logkit.Fields{"recover": fmt.Sprint(r)})
			}
		}()

		uc.deleteFromStorage(context.Background(), fullPath)
		uc.deleteFromStorage(context.Background(), thumbPath)
	})
}

func (uc *AvatarUseCase) UploadUserAvatar(ctx context.Context, userID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("UploadUserAvatar - GetByID: %w", err)
	}

	if user.IsBanned {
		return "", "", httperr.ErrUserBanned
	}

	processed, err := uc.processor.Process(file)
	if err != nil {
		return "", "", fmt.Errorf("UploadUserAvatar - Process: %w", err)
	}

	fullPath := uc.buildStoragePath(domain.AvatarEntityUser, userID, processed.SHA256Hash, "full")
	thumbPath := uc.buildStoragePath(domain.AvatarEntityUser, userID, processed.SHA256Hash, "thumb")

	if err := uc.deps.Storage.Upload(ctx, fullPath, io.NopCloser(NewBytesReader(processed.FullImage)), int64(len(processed.FullImage)), "image/webp"); err != nil {
		return "", "", fmt.Errorf("UploadUserAvatar - Upload full: %w", err)
	}

	if err := uc.deps.Storage.Upload(ctx, thumbPath, io.NopCloser(NewBytesReader(processed.ThumbnailImage)), int64(len(processed.ThumbnailImage)), "image/webp"); err != nil {
		uc.deleteFromStorage(ctx, fullPath)

		return "", "", fmt.Errorf("UploadUserAvatar - Upload thumb: %w", err)
	}

	if err := uc.deps.UserRepo.UpdateAvatarURL(ctx, userID, fullPath); err != nil {
		uc.deleteFromStorage(ctx, fullPath)
		uc.deleteFromStorage(ctx, thumbPath)

		return "", "", fmt.Errorf("UploadUserAvatar - UpdateAvatarURL: %w", err)
	}

	if user.AvatarURL != nil && *user.AvatarURL != "" {
		uc.goDeleteOldAvatar(*user.AvatarURL, thumbPathFromFull(*user.AvatarURL))
	}

	uc.invalidateCache(&userID, nil)

	fullURL, err = uc.deps.Storage.GetPresignedURL(ctx, fullPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return "", "", fmt.Errorf("UploadUserAvatar - GetPresignedURL full: %w", err)
	}

	thumbURL, err = uc.deps.Storage.GetPresignedURL(ctx, thumbPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return "", "", fmt.Errorf("UploadUserAvatar - GetPresignedURL thumb: %w", err)
	}

	return fullURL, thumbURL, nil
}

func (uc *AvatarUseCase) DeleteUserAvatar(ctx context.Context, userID uuid.UUID) error {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("DeleteUserAvatar - GetByID: %w", err)
	}

	oldAvatarURL := user.AvatarURL

	_, err = uc.deps.UserRepo.ClearAvatarURL(ctx, userID)
	if err != nil {
		return fmt.Errorf("DeleteUserAvatar - ClearAvatarURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" {
		uc.goDeleteOldAvatar(*oldAvatarURL, thumbPathFromFull(*oldAvatarURL))
	}

	uc.invalidateCache(&userID, nil)

	return nil
}

func (uc *AvatarUseCase) GetUserAvatarURL(ctx context.Context, userID uuid.UUID) (fullURL, thumbURL *string, err error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("GetUserAvatarURL - GetByID: %w", err)
	}

	if user.AvatarURL == nil || *user.AvatarURL == "" {
		return nil, nil, nil
	}

	cacheKey := cache.KeyAvatarUser(userID.String())
	if rawCache, err := uc.deps.Cache.Get(ctx, cacheKey); err == nil {
		var cached struct {
			FullURL  string `json:"full"`
			ThumbURL string `json:"thumb"`
		}
		if json.Unmarshal([]byte(rawCache), &cached) == nil {
			return &cached.FullURL, &cached.ThumbURL, nil
		}
	}

	fullPath := *user.AvatarURL
	thumbPath := thumbPathFromFull(fullPath)

	fullPresigned, err := uc.deps.Storage.GetPresignedURL(ctx, fullPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("GetUserAvatarURL - GetPresignedURL full: %w", err)
	}

	thumbPresigned, err := uc.deps.Storage.GetPresignedURL(ctx, thumbPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("GetUserAvatarURL - GetPresignedURL thumb: %w", err)
	}

	cached := struct {
		FullURL  string `json:"full"`
		ThumbURL string `json:"thumb"`
	}{FullURL: fullPresigned, ThumbURL: thumbPresigned}
	if cacheBytes, _ := json.Marshal(cached); cacheBytes != nil {
		_ = uc.deps.Cache.Set(ctx, cacheKey, cacheBytes, uc.deps.Config.CacheTTL)
	}

	return &fullPresigned, &thumbPresigned, nil
}

func (uc *AvatarUseCase) UploadTeamAvatar(ctx context.Context, teamID, callerID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return "", "", fmt.Errorf("UploadTeamAvatar - GetByID: %w", err)
	}

	if team.CaptainID != callerID {
		return "", "", httperr.ErrNotTeamCaptain
	}

	if team.IsBanned {
		return "", "", httperr.ErrTeamBanned
	}

	processed, err := uc.processor.Process(file)
	if err != nil {
		return "", "", fmt.Errorf("UploadTeamAvatar - Process: %w", err)
	}

	fullPath := uc.buildStoragePath(domain.AvatarEntityTeam, teamID, processed.SHA256Hash, "full")
	thumbPath := uc.buildStoragePath(domain.AvatarEntityTeam, teamID, processed.SHA256Hash, "thumb")

	if err := uc.deps.Storage.Upload(ctx, fullPath, io.NopCloser(NewBytesReader(processed.FullImage)), int64(len(processed.FullImage)), "image/webp"); err != nil {
		return "", "", fmt.Errorf("UploadTeamAvatar - Upload full: %w", err)
	}

	if err := uc.deps.Storage.Upload(ctx, thumbPath, io.NopCloser(NewBytesReader(processed.ThumbnailImage)), int64(len(processed.ThumbnailImage)), "image/webp"); err != nil {
		uc.deleteFromStorage(ctx, fullPath)

		return "", "", fmt.Errorf("UploadTeamAvatar - Upload thumb: %w", err)
	}

	if err := uc.deps.TeamRepo.UpdateAvatarURL(ctx, teamID, fullPath); err != nil {
		uc.deleteFromStorage(ctx, fullPath)
		uc.deleteFromStorage(ctx, thumbPath)

		return "", "", fmt.Errorf("UploadTeamAvatar - UpdateAvatarURL: %w", err)
	}

	if team.AvatarURL != nil && *team.AvatarURL != "" {
		uc.goDeleteOldAvatar(*team.AvatarURL, thumbPathFromFull(*team.AvatarURL))
	}

	uc.invalidateCache(nil, &teamID)

	fullURL, err = uc.deps.Storage.GetPresignedURL(ctx, fullPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return "", "", fmt.Errorf("UploadTeamAvatar - GetPresignedURL full: %w", err)
	}

	thumbURL, err = uc.deps.Storage.GetPresignedURL(ctx, thumbPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return "", "", fmt.Errorf("UploadTeamAvatar - GetPresignedURL thumb: %w", err)
	}

	return fullURL, thumbURL, nil
}

func (uc *AvatarUseCase) DeleteTeamAvatar(ctx context.Context, teamID, callerID uuid.UUID) error {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("DeleteTeamAvatar - GetByID: %w", err)
	}

	if team.CaptainID != callerID {
		return fmt.Errorf("DeleteTeamAvatar: %w", httperr.ErrNotTeamCaptain)
	}

	oldAvatarURL := team.AvatarURL

	_, err = uc.deps.TeamRepo.ClearAvatarURL(ctx, teamID)
	if err != nil {
		return fmt.Errorf("DeleteTeamAvatar - ClearAvatarURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" {
		uc.goDeleteOldAvatar(*oldAvatarURL, thumbPathFromFull(*oldAvatarURL))
	}

	uc.invalidateCache(nil, &teamID)

	return nil
}

func (uc *AvatarUseCase) GetTeamAvatarURL(ctx context.Context, teamID uuid.UUID) (fullURL, thumbURL *string, err error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, nil, fmt.Errorf("GetTeamAvatarURL - GetByID: %w", err)
	}

	if team.AvatarURL == nil || *team.AvatarURL == "" {
		return nil, nil, nil
	}

	cacheKey := cache.KeyAvatarTeam(teamID.String())
	if rawCache, err := uc.deps.Cache.Get(ctx, cacheKey); err == nil {
		var cached struct {
			FullURL  string `json:"full"`
			ThumbURL string `json:"thumb"`
		}
		if json.Unmarshal([]byte(rawCache), &cached) == nil {
			return &cached.FullURL, &cached.ThumbURL, nil
		}
	}

	fullPath := *team.AvatarURL
	thumbPath := thumbPathFromFull(fullPath)

	fullPresigned, err := uc.deps.Storage.GetPresignedURL(ctx, fullPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("GetTeamAvatarURL - GetPresignedURL full: %w", err)
	}

	thumbPresigned, err := uc.deps.Storage.GetPresignedURL(ctx, thumbPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("GetTeamAvatarURL - GetPresignedURL thumb: %w", err)
	}

	cached := struct {
		FullURL  string `json:"full"`
		ThumbURL string `json:"thumb"`
	}{FullURL: fullPresigned, ThumbURL: thumbPresigned}
	if cacheBytes, _ := json.Marshal(cached); cacheBytes != nil {
		_ = uc.deps.Cache.Set(ctx, cacheKey, cacheBytes, uc.deps.Config.CacheTTL)
	}

	return &fullPresigned, &thumbPresigned, nil
}

func (uc *AvatarUseCase) AdminUploadUserAvatar(ctx context.Context, userID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("AdminUploadUserAvatar - GetByID: %w", err)
	}

	fullURL, thumbURL, err = uc.uploadAvatar(ctx, domain.AvatarEntityUser, userID, file, user.AvatarURL,
		func(ctx context.Context, id uuid.UUID, path string) error {
			return uc.deps.UserRepo.UpdateAvatarURL(ctx, id, path)
		},
		func(id uuid.UUID) { uc.invalidateCache(&id, nil) },
	)
	if err != nil {
		return "", "", err
	}

	return fullURL, thumbURL, nil
}

func (uc *AvatarUseCase) AdminDeleteUserAvatar(ctx context.Context, userID uuid.UUID) error {
	if _, err := uc.deps.UserRepo.GetByID(ctx, userID); err != nil {
		return fmt.Errorf("AdminDeleteUserAvatar - GetByID: %w", err)
	}

	oldURL, err := uc.deps.UserRepo.ClearAvatarURL(ctx, userID)
	if err != nil {
		return fmt.Errorf("AdminDeleteUserAvatar - ClearAvatarURL: %w", err)
	}

	if oldURL != nil && *oldURL != "" {
		uc.goDeleteOldAvatar(*oldURL, thumbPathFromFull(*oldURL))
	}

	uc.invalidateCache(&userID, nil)

	return nil
}

func (uc *AvatarUseCase) AdminUploadTeamAvatar(ctx context.Context, teamID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return "", "", fmt.Errorf("AdminUploadTeamAvatar - GetByID: %w", err)
	}

	fullURL, thumbURL, err = uc.uploadAvatar(ctx, domain.AvatarEntityTeam, teamID, file, team.AvatarURL,
		func(ctx context.Context, id uuid.UUID, path string) error {
			return uc.deps.TeamRepo.UpdateAvatarURL(ctx, id, path)
		},
		func(id uuid.UUID) { uc.invalidateCache(nil, &id) },
	)
	if err != nil {
		return "", "", err
	}

	return fullURL, thumbURL, nil
}

func (uc *AvatarUseCase) AdminDeleteTeamAvatar(ctx context.Context, teamID uuid.UUID) error {
	if _, err := uc.deps.TeamRepo.GetByID(ctx, teamID); err != nil {
		return fmt.Errorf("AdminDeleteTeamAvatar - GetByID: %w", err)
	}

	oldURL, err := uc.deps.TeamRepo.ClearAvatarURL(ctx, teamID)
	if err != nil {
		return fmt.Errorf("AdminDeleteTeamAvatar - ClearAvatarURL: %w", err)
	}

	if oldURL != nil && *oldURL != "" {
		uc.goDeleteOldAvatar(*oldURL, thumbPathFromFull(*oldURL))
	}

	uc.invalidateCache(nil, &teamID)

	return nil
}

func (uc *AvatarUseCase) uploadAvatar(
	ctx context.Context,
	entityType domain.AvatarEntityType,
	entityID uuid.UUID,
	file io.Reader,
	oldAvatarURL *string,
	updateURL func(ctx context.Context, id uuid.UUID, path string) error,
	invalidate func(uuid.UUID),
) (fullURL, thumbURL string, err error) {
	processed, err := uc.processor.Process(file)
	if err != nil {
		return "", "", fmt.Errorf("uploadAvatar - Process: %w", err)
	}

	fullPath := uc.buildStoragePath(entityType, entityID, processed.SHA256Hash, "full")
	thumbPath := uc.buildStoragePath(entityType, entityID, processed.SHA256Hash, "thumb")

	if err := uc.deps.Storage.Upload(ctx, fullPath, io.NopCloser(NewBytesReader(processed.FullImage)), int64(len(processed.FullImage)), "image/webp"); err != nil {
		return "", "", fmt.Errorf("uploadAvatar - Upload full: %w", err)
	}

	if err := uc.deps.Storage.Upload(ctx, thumbPath, io.NopCloser(NewBytesReader(processed.ThumbnailImage)), int64(len(processed.ThumbnailImage)), "image/webp"); err != nil {
		uc.deleteFromStorage(ctx, fullPath)

		return "", "", fmt.Errorf("uploadAvatar - Upload thumb: %w", err)
	}

	if err := updateURL(ctx, entityID, fullPath); err != nil {
		uc.deleteFromStorage(ctx, fullPath)
		uc.deleteFromStorage(ctx, thumbPath)

		return "", "", fmt.Errorf("uploadAvatar - UpdateURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" && *oldAvatarURL != fullPath {
		uc.goDeleteOldAvatar(*oldAvatarURL, thumbPathFromFull(*oldAvatarURL))
	}

	invalidate(entityID)

	fullURL, err = uc.deps.Storage.GetPresignedURL(ctx, fullPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return "", "", fmt.Errorf("uploadAvatar - GetPresignedURL full: %w", err)
	}

	thumbURL, err = uc.deps.Storage.GetPresignedURL(ctx, thumbPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return "", "", fmt.Errorf("uploadAvatar - GetPresignedURL thumb: %w", err)
	}

	return fullURL, thumbURL, nil
}

func (uc *AvatarUseCase) buildStoragePath(entityType domain.AvatarEntityType, entityID uuid.UUID, hash, variant string) string {
	return fmt.Sprintf("%s/%s/%s_%s.webp", entityType, entityID.String(), hash, variant)
}

func thumbPathFromFull(fullPath string) string {
	if len(fullPath) < 10 || !strings.HasSuffix(fullPath, "_full.webp") {
		return ""
	}

	return fullPath[:len(fullPath)-10] + "_thumb.webp"
}

func (uc *AvatarUseCase) deleteFromStorage(ctx context.Context, path string) {
	if path == "" {
		return
	}

	err := uc.deps.Storage.Delete(ctx, path)
	if err != nil {
		uc.deps.Logger.WithError(err).Warn("AvatarUseCase - failed to delete file from storage", logkit.Fields{"path": path})
	}
}

func (uc *AvatarUseCase) invalidateCache(userID, teamID *uuid.UUID) {
	if userID != nil {
		_ = uc.deps.Cache.Del(context.Background(), cache.KeyAvatarUser(userID.String()))
	}

	if teamID != nil {
		_ = uc.deps.Cache.Del(context.Background(), cache.KeyAvatarTeam(teamID.String()))
	}
}
