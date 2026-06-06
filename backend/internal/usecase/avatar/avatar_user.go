package avatar

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

func (uc *AvatarUseCase) UploadUserAvatar(ctx context.Context, userID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("AvatarUseCase - UploadUserAvatar - GetByID: %w", err)
	}

	if user.IsBanned {
		return "", "", apperr.ErrUserBanned
	}

	return uc.uploadAvatar(ctx, domain.AvatarEntityUser, userID, file,
		func(ctx context.Context, path string) (*string, error) {
			return uc.updateUserAvatarURL(ctx, userID, path, func(user *domain.User) error {
				if user.IsBanned {
					return apperr.ErrUserBanned
				}

				return nil
			})
		},
		func(ctx context.Context) { uc.invalidateCache(ctx, &userID, nil) },
	)
}

func (uc *AvatarUseCase) DeleteUserAvatar(ctx context.Context, userID uuid.UUID) error {
	oldAvatarURL, err := uc.clearUserAvatarURL(ctx, userID, nil)
	if err != nil {
		return fmt.Errorf("AvatarUseCase - DeleteUserAvatar - clearUserAvatarURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" {
		uc.goDeleteOldAvatar(ctx, *oldAvatarURL, domain.ThumbPathFromFull(*oldAvatarURL))
	}

	uc.invalidateCache(ctx, &userID, nil)

	return nil
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

	return uc.resolvePresignedURLs(ctx, cacheutil.KeyAvatarUser(userID.String()), user.AvatarURL)
}
