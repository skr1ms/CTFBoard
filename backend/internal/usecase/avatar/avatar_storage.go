package avatar

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

func newBytesReader(data []byte) io.Reader {
	return bytes.NewReader(data)
}

func cloneStringPtr(v *string) *string {
	if v == nil {
		return nil
	}

	copied := *v

	return &copied
}

// goDeleteOldAvatar asynchronously deletes the previous full and thumbnail
// avatar files from storage. It detaches from the originating request cancellation
// so cleanup can finish after a successful DB update. A deferred recover prevents
// a storage panic from crashing the server; errors are only logged.
func (uc *AvatarUseCase) goDeleteOldAvatar(ctx context.Context, fullPath, thumbPath string) {
	uc.wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				uc.deps.Logger.Error("AvatarUseCase - goDeleteOldAvatar panic", logkit.Fields{"recover": fmt.Sprint(r)})
			}
		}()

		deleteCtx, cancel := uc.sideEffectCtx(ctx)
		defer cancel()

		uc.deleteFromStorage(deleteCtx, fullPath)
		uc.deleteFromStorage(deleteCtx, thumbPath)
	})
}

// uploadAvatar is the generic avatar upload helper shared by Admin* methods.
// It runs Process, uploads full+thumbnail (rolling back uploaded objects on any
// subsequent failure), calls persistURL to persist the storage path, enqueues
// async deletion of the old avatar when the committed old URL differs from the new path,
// invokes the caller-supplied invalidate function, and returns pre-signed URLs.
func (uc *AvatarUseCase) uploadAvatar(
	ctx context.Context,
	entityType domain.AvatarEntityType,
	entityID uuid.UUID,
	file io.Reader,
	persistURL func(ctx context.Context, path string) (*string, error),
	invalidate func(context.Context),
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

	oldAvatarURL, err := persistURL(ctx, fullPath)
	if err != nil {
		uc.deleteFromStorage(ctx, fullPath)
		uc.deleteFromStorage(ctx, thumbPath)

		return "", "", fmt.Errorf("AvatarUseCase - uploadAvatar - persistURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" && *oldAvatarURL != fullPath {
		uc.goDeleteOldAvatar(ctx, *oldAvatarURL, domain.ThumbPathFromFull(*oldAvatarURL))
	}

	invalidate(ctx)

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

func (uc *AvatarUseCase) invalidateCache(ctx context.Context, userID, teamID *uuid.UUID) {
	ctx, cancel := uc.sideEffectCtx(ctx)
	defer cancel()

	if userID != nil {
		_ = uc.deps.Cache.Del(ctx, cacheutil.KeyAvatarUser(userID.String()))
	}

	if teamID != nil {
		_ = uc.deps.Cache.Del(ctx, cacheutil.KeyAvatarTeam(teamID.String()))
	}
}

func (uc *AvatarUseCase) updateUserAvatarURL(ctx context.Context, userID uuid.UUID, path string, validate func(*domain.User) error) (*string, error) {
	var oldAvatarURL *string

	useTransaction := uc.deps.TM != nil

	if err := uc.runAvatarMutation(ctx, func(ctx context.Context) error {
		if useTransaction {
			if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
				return fmt.Errorf("AvatarUseCase - updateUserAvatarURL - Lock: %w", err)
			}
		}

		user, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("AvatarUseCase - updateUserAvatarURL - GetByID: %w", err)
		}

		if validate != nil {
			if err := validate(user); err != nil {
				return err
			}
		}

		oldAvatarURL = cloneStringPtr(user.AvatarURL)

		if err := uc.deps.UserRepo.UpdateAvatarURL(ctx, userID, path); err != nil {
			return fmt.Errorf("AvatarUseCase - updateUserAvatarURL - UpdateAvatarURL: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return oldAvatarURL, nil
}

func (uc *AvatarUseCase) clearUserAvatarURL(ctx context.Context, userID uuid.UUID, validate func(*domain.User) error) (*string, error) {
	var oldAvatarURL *string

	useTransaction := uc.deps.TM != nil

	if err := uc.runAvatarMutation(ctx, func(ctx context.Context) error {
		if useTransaction {
			if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
				return fmt.Errorf("AvatarUseCase - clearUserAvatarURL - Lock: %w", err)
			}
		}

		user, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("AvatarUseCase - clearUserAvatarURL - GetByID: %w", err)
		}

		if validate != nil {
			if err := validate(user); err != nil {
				return err
			}
		}

		oldAvatarURL = cloneStringPtr(user.AvatarURL)

		if err := uc.deps.UserRepo.ClearAvatarURL(ctx, userID); err != nil {
			return fmt.Errorf("AvatarUseCase - clearUserAvatarURL - ClearAvatarURL: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return oldAvatarURL, nil
}

func (uc *AvatarUseCase) updateTeamAvatarURL(ctx context.Context, teamID uuid.UUID, path string, validate func(*domain.Team) error) (*string, error) {
	var oldAvatarURL *string

	useTransaction := uc.deps.TM != nil

	if err := uc.runAvatarMutation(ctx, func(ctx context.Context) error {
		if useTransaction {
			if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
				return fmt.Errorf("AvatarUseCase - updateTeamAvatarURL - Lock: %w", err)
			}
		}

		team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("AvatarUseCase - updateTeamAvatarURL - GetByID: %w", err)
		}

		if validate != nil {
			if err := validate(team); err != nil {
				return err
			}
		}

		oldAvatarURL = cloneStringPtr(team.AvatarURL)

		if err := uc.deps.TeamRepo.UpdateAvatarURL(ctx, teamID, path); err != nil {
			return fmt.Errorf("AvatarUseCase - updateTeamAvatarURL - UpdateAvatarURL: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return oldAvatarURL, nil
}

func (uc *AvatarUseCase) clearTeamAvatarURL(ctx context.Context, teamID uuid.UUID, validate func(*domain.Team) error) (*string, error) {
	var oldAvatarURL *string

	useTransaction := uc.deps.TM != nil

	if err := uc.runAvatarMutation(ctx, func(ctx context.Context) error {
		if useTransaction {
			if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
				return fmt.Errorf("AvatarUseCase - clearTeamAvatarURL - Lock: %w", err)
			}
		}

		team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("AvatarUseCase - clearTeamAvatarURL - GetByID: %w", err)
		}

		if validate != nil {
			if err := validate(team); err != nil {
				return err
			}
		}

		oldAvatarURL = cloneStringPtr(team.AvatarURL)

		if err := uc.deps.TeamRepo.ClearAvatarURL(ctx, teamID); err != nil {
			return fmt.Errorf("AvatarUseCase - clearTeamAvatarURL - ClearAvatarURL: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return oldAvatarURL, nil
}
