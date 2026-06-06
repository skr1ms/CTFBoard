package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *UserRepo) UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string) error {
	err := r.Q(ctx).UpdateUserAvatarURL(ctx, sqlc.UpdateUserAvatarURLParams{
		ID:        userID,
		AvatarUrl: &avatarURL,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateAvatarURL: %w", err)
	}

	return nil
}

func (r *UserRepo) ClearAvatarURL(ctx context.Context, userID uuid.UUID) error {
	if err := r.Q(ctx).ClearUserAvatarURL(ctx, userID); err != nil {
		return fmt.Errorf("UserRepo - ClearAvatarURL: %w", err)
	}

	return nil
}

func (r *UserRepo) ListAllUserAvatarURLs(ctx context.Context) ([]*string, error) {
	urls, err := r.Q(ctx).ListAllUserAvatarURLs(ctx)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - ListAllUserAvatarURLs: %w", err)
	}

	return urls, nil
}
