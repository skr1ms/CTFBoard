package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *TeamRepo) UpdateAvatarURL(ctx context.Context, teamID uuid.UUID, avatarURL string) error {
	err := r.Q(ctx).UpdateTeamAvatarURL(ctx, sqlc.UpdateTeamAvatarURLParams{
		ID:        teamID,
		AvatarUrl: &avatarURL,
	})
	if err != nil {
		return fmt.Errorf("TeamRepo - UpdateAvatarURL: %w", err)
	}

	return nil
}

func (r *TeamRepo) ClearAvatarURL(ctx context.Context, teamID uuid.UUID) error {
	if err := r.Q(ctx).ClearTeamAvatarURL(ctx, teamID); err != nil {
		return fmt.Errorf("TeamRepo - ClearAvatarURL: %w", err)
	}

	return nil
}

func (r *TeamRepo) ListAllTeamAvatarURLs(ctx context.Context) ([]*string, error) {
	urls, err := r.Q(ctx).ListAllTeamAvatarURLs(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamRepo - ListAllTeamAvatarURLs: %w", err)
	}

	return urls, nil
}
