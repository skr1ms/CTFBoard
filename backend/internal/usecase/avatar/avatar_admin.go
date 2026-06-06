package avatar

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func (uc *AvatarUseCase) AdminUploadUserAvatar(ctx context.Context, userID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error) {
	if _, err := uc.deps.UserRepo.GetByID(ctx, userID); err != nil {
		return "", "", fmt.Errorf("AvatarUseCase - AdminUploadUserAvatar - GetByID: %w", err)
	}

	fullURL, thumbURL, err = uc.uploadAvatar(ctx, domain.AvatarEntityUser, userID, file,
		func(ctx context.Context, path string) (*string, error) {
			return uc.updateUserAvatarURL(ctx, userID, path, nil)
		},
		func(ctx context.Context) { uc.invalidateCache(ctx, &userID, nil) },
	)
	if err != nil {
		return "", "", err
	}

	return fullURL, thumbURL, nil
}

func (uc *AvatarUseCase) AdminDeleteUserAvatar(ctx context.Context, userID uuid.UUID) error {
	oldAvatarURL, err := uc.clearUserAvatarURL(ctx, userID, nil)
	if err != nil {
		return fmt.Errorf("AvatarUseCase - AdminDeleteUserAvatar - clearUserAvatarURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" {
		uc.goDeleteOldAvatar(ctx, *oldAvatarURL, domain.ThumbPathFromFull(*oldAvatarURL))
	}

	uc.invalidateCache(ctx, &userID, nil)

	return nil
}

func (uc *AvatarUseCase) AdminUploadTeamAvatar(ctx context.Context, teamID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error) {
	if _, err := uc.deps.TeamRepo.GetByID(ctx, teamID); err != nil {
		return "", "", fmt.Errorf("AvatarUseCase - AdminUploadTeamAvatar - GetByID: %w", err)
	}

	fullURL, thumbURL, err = uc.uploadAvatar(ctx, domain.AvatarEntityTeam, teamID, file,
		func(ctx context.Context, path string) (*string, error) {
			return uc.updateTeamAvatarURL(ctx, teamID, path, nil)
		},
		func(ctx context.Context) { uc.invalidateCache(ctx, nil, &teamID) },
	)
	if err != nil {
		return "", "", err
	}

	return fullURL, thumbURL, nil
}

func (uc *AvatarUseCase) AdminDeleteTeamAvatar(ctx context.Context, teamID uuid.UUID) error {
	oldAvatarURL, err := uc.clearTeamAvatarURL(ctx, teamID, nil)
	if err != nil {
		return fmt.Errorf("AvatarUseCase - AdminDeleteTeamAvatar - clearTeamAvatarURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" {
		uc.goDeleteOldAvatar(ctx, *oldAvatarURL, domain.ThumbPathFromFull(*oldAvatarURL))
	}

	uc.invalidateCache(ctx, nil, &teamID)

	return nil
}
