package avatar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const (
	contentTypeWebP                = "image/webp"
	resolveAvatarsBatchConcurrency = 10
)

func newBytesReader(data []byte) io.Reader {
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
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &AvatarUseCase{
		deps:      deps,
		processor: NewImageProcessor(deps.Config),
	}
}

func (uc *AvatarUseCase) Wait() { uc.wg.Wait() }

// goDeleteOldAvatar asynchronously deletes the previous full and thumbnail
// avatar files from storage. It runs on context.Background() so it survives
// cancellation of the originating request context. A deferred recover prevents
// a storage panic from crashing the server; errors are only logged.
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
		return "", "", fmt.Errorf("AvatarUseCase - UploadUserAvatar - GetByID: %w", err)
	}

	if user.IsBanned {
		return "", "", apperr.ErrUserBanned
	}

	return uc.uploadAvatar(ctx, domain.AvatarEntityUser, userID, file, user.AvatarURL,
		func(ctx context.Context, id uuid.UUID, path string) error {
			return uc.deps.UserRepo.UpdateAvatarURL(ctx, id, path)
		},
		func(id uuid.UUID) { uc.invalidateCache(&id, nil) },
	)
}

func (uc *AvatarUseCase) DeleteUserAvatar(ctx context.Context, userID uuid.UUID) error {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("AvatarUseCase - DeleteUserAvatar - GetByID: %w", err)
	}

	oldAvatarURL := user.AvatarURL

	if err = uc.deps.UserRepo.ClearAvatarURL(ctx, userID); err != nil {
		return fmt.Errorf("AvatarUseCase - DeleteUserAvatar - ClearAvatarURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" {
		uc.goDeleteOldAvatar(*oldAvatarURL, domain.ThumbPathFromFull(*oldAvatarURL))
	}

	uc.invalidateCache(&userID, nil)

	return nil
}

// resolvePresignedURLs checks the Redis cache for pre-signed avatar URLs and,
// on a miss, fetches them from storage and writes the result back to cache.
// avatarURL must be non-nil and non-empty (callers are responsible for the guard).
func (uc *AvatarUseCase) resolvePresignedURLs(ctx context.Context, cacheKey string, avatarURL *string) (fullURL, thumbURL *string, err error) {
	if rawCache, err := uc.deps.Cache.Get(ctx, cacheKey); err == nil {
		var cached struct {
			FullURL  string `json:"full"`
			ThumbURL string `json:"thumb"`
		}
		if json.Unmarshal([]byte(rawCache), &cached) == nil {
			return &cached.FullURL, &cached.ThumbURL, nil
		}
	}

	fullPath := *avatarURL
	thumbPath := domain.ThumbPathFromFull(fullPath)

	fullPresigned, err := uc.deps.Storage.GetPresignedURL(ctx, fullPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("AvatarUseCase - resolvePresignedURLs - GetPresignedURL full: %w", err)
	}

	thumbPresigned, err := uc.deps.Storage.GetPresignedURL(ctx, thumbPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("AvatarUseCase - resolvePresignedURLs - GetPresignedURL thumb: %w", err)
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

// GetUserAvatarURL returns pre-signed URLs for the user's full and thumbnail
// avatar images. It uses a two-layer cache: a Redis JSON entry is checked first;
// on miss, pre-signed URLs are fetched from storage and written back to Redis
// for CacheTTL. Returns three nil values when the user has no avatar set.
func (uc *AvatarUseCase) GetUserAvatarURL(ctx context.Context, userID uuid.UUID) (fullURL, thumbURL *string, err error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("AvatarUseCase - GetUserAvatarURL - GetByID: %w", err)
	}

	if user.AvatarURL == nil || *user.AvatarURL == "" {
		return nil, nil, nil
	}

	return uc.resolvePresignedURLs(ctx, cache.KeyAvatarUser(userID.String()), user.AvatarURL)
}

func (uc *AvatarUseCase) UploadTeamAvatar(ctx context.Context, teamID, callerID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return "", "", fmt.Errorf("AvatarUseCase - UploadTeamAvatar - GetByID: %w", err)
	}

	if team.CaptainID != callerID {
		return "", "", apperr.ErrNotTeamCaptain
	}

	if team.IsBanned {
		return "", "", apperr.ErrTeamBanned
	}

	return uc.uploadAvatar(ctx, domain.AvatarEntityTeam, teamID, file, team.AvatarURL,
		func(ctx context.Context, id uuid.UUID, path string) error {
			return uc.deps.TeamRepo.UpdateAvatarURL(ctx, id, path)
		},
		func(id uuid.UUID) { uc.invalidateCache(nil, &id) },
	)
}

func (uc *AvatarUseCase) DeleteTeamAvatar(ctx context.Context, teamID, callerID uuid.UUID) error {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("AvatarUseCase - DeleteTeamAvatar - GetByID: %w", err)
	}

	if team.CaptainID != callerID {
		return apperr.ErrNotTeamCaptain
	}

	oldAvatarURL := team.AvatarURL

	if err = uc.deps.TeamRepo.ClearAvatarURL(ctx, teamID); err != nil {
		return fmt.Errorf("AvatarUseCase - DeleteTeamAvatar - ClearAvatarURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" {
		uc.goDeleteOldAvatar(*oldAvatarURL, domain.ThumbPathFromFull(*oldAvatarURL))
	}

	uc.invalidateCache(nil, &teamID)

	return nil
}

// GetTeamAvatarURL returns pre-signed URLs for the team's full and thumbnail
// avatar images using the same two-layer Redis cache pattern as GetUserAvatarURL.
// Returns three nil values when the team has no avatar set.
func (uc *AvatarUseCase) GetTeamAvatarURL(ctx context.Context, teamID uuid.UUID) (fullURL, thumbURL *string, err error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, nil, fmt.Errorf("AvatarUseCase - GetTeamAvatarURL - GetByID: %w", err)
	}

	if team.AvatarURL == nil || *team.AvatarURL == "" {
		return nil, nil, nil
	}

	return uc.resolvePresignedURLs(ctx, cache.KeyAvatarTeam(teamID.String()), team.AvatarURL)
}

// GetTeamAvatarURLBatch fetches thumbnail avatar URLs for a batch of teams in a
// single DB round-trip. Only teams that have an avatar set are included in the
// returned map. The map key is the team ID; the value is the pre-signed thumb URL.
func (uc *AvatarUseCase) GetTeamAvatarURLBatch(ctx context.Context, teamIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(teamIDs) == 0 {
		return nil, nil
	}

	teams, err := uc.deps.TeamRepo.GetByIDs(ctx, teamIDs)
	if err != nil {
		return nil, fmt.Errorf("AvatarUseCase - GetTeamAvatarURLBatch - GetByIDs: %w", err)
	}

	result := make(map[uuid.UUID]string, len(teams))

	var mu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(resolveAvatarsBatchConcurrency)

	for teamID, team := range teams {
		if team.AvatarURL == nil || *team.AvatarURL == "" {
			continue
		}

		g.Go(func() error {
			_, thumbURL, resolveErr := uc.resolvePresignedURLs(gctx, cache.KeyAvatarTeam(teamID.String()), team.AvatarURL)
			if resolveErr != nil {
				return resolveErr
			}

			if thumbURL != nil && *thumbURL != "" {
				mu.Lock()
				result[teamID] = *thumbURL
				mu.Unlock()
			}

			return nil
		})
	}

	if err = g.Wait(); err != nil {
		return nil, fmt.Errorf("AvatarUseCase - GetTeamAvatarURLBatch - resolve: %w", err)
	}

	return result, nil
}

func (uc *AvatarUseCase) AdminUploadUserAvatar(ctx context.Context, userID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("AvatarUseCase - AdminUploadUserAvatar - GetByID: %w", err)
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
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("AvatarUseCase - AdminDeleteUserAvatar - GetByID: %w", err)
	}

	oldAvatarURL := user.AvatarURL

	if err = uc.deps.UserRepo.ClearAvatarURL(ctx, userID); err != nil {
		return fmt.Errorf("AvatarUseCase - AdminDeleteUserAvatar - ClearAvatarURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" {
		uc.goDeleteOldAvatar(*oldAvatarURL, domain.ThumbPathFromFull(*oldAvatarURL))
	}

	uc.invalidateCache(&userID, nil)

	return nil
}

func (uc *AvatarUseCase) AdminUploadTeamAvatar(ctx context.Context, teamID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error) {
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return "", "", fmt.Errorf("AvatarUseCase - AdminUploadTeamAvatar - GetByID: %w", err)
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
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("AvatarUseCase - AdminDeleteTeamAvatar - GetByID: %w", err)
	}

	oldAvatarURL := team.AvatarURL

	if err = uc.deps.TeamRepo.ClearAvatarURL(ctx, teamID); err != nil {
		return fmt.Errorf("AvatarUseCase - AdminDeleteTeamAvatar - ClearAvatarURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" {
		uc.goDeleteOldAvatar(*oldAvatarURL, domain.ThumbPathFromFull(*oldAvatarURL))
	}

	uc.invalidateCache(nil, &teamID)

	return nil
}

// uploadAvatar is the generic avatar upload helper shared by Admin* methods.
// It runs Process, uploads full+thumbnail (rolling back uploaded objects on any
// subsequent failure), calls updateURL to persist the storage path, enqueues
// async deletion of the old avatar when oldAvatarURL differs from the new path,
// invokes the caller-supplied invalidate function, and returns pre-signed URLs.
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
		return "", "", fmt.Errorf("AvatarUseCase - uploadAvatar - Process: %w", err)
	}

	fullPath := uc.buildStoragePath(entityType, entityID, processed.SHA256Hash, "full")
	thumbPath := uc.buildStoragePath(entityType, entityID, processed.SHA256Hash, "thumb")

	if err := uc.deps.Storage.Upload(ctx, fullPath, io.NopCloser(newBytesReader(processed.FullImage)), int64(len(processed.FullImage)), contentTypeWebP); err != nil {
		return "", "", fmt.Errorf("AvatarUseCase - uploadAvatar - Upload full: %w", err)
	}

	if err := uc.deps.Storage.Upload(ctx, thumbPath, io.NopCloser(newBytesReader(processed.ThumbnailImage)), int64(len(processed.ThumbnailImage)), contentTypeWebP); err != nil {
		uc.deleteFromStorage(ctx, fullPath)

		return "", "", fmt.Errorf("AvatarUseCase - uploadAvatar - Upload thumb: %w", err)
	}

	if err := updateURL(ctx, entityID, fullPath); err != nil {
		uc.deleteFromStorage(ctx, fullPath)
		uc.deleteFromStorage(ctx, thumbPath)

		return "", "", fmt.Errorf("AvatarUseCase - uploadAvatar - UpdateURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" && *oldAvatarURL != fullPath {
		uc.goDeleteOldAvatar(*oldAvatarURL, domain.ThumbPathFromFull(*oldAvatarURL))
	}

	invalidate(entityID)

	fullURL, err = uc.deps.Storage.GetPresignedURL(ctx, fullPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return "", "", fmt.Errorf("AvatarUseCase - uploadAvatar - GetPresignedURL full: %w", err)
	}

	thumbURL, err = uc.deps.Storage.GetPresignedURL(ctx, thumbPath, uc.deps.Config.PresignedTTL)
	if err != nil {
		return "", "", fmt.Errorf("AvatarUseCase - uploadAvatar - GetPresignedURL thumb: %w", err)
	}

	return fullURL, thumbURL, nil
}

func (uc *AvatarUseCase) buildStoragePath(entityType domain.AvatarEntityType, entityID uuid.UUID, hash, variant string) string {
	return fmt.Sprintf("%s/%s/%s_%s.webp", entityType, entityID.String(), hash, variant)
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
