package avatar

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

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

	return uc.uploadAvatar(ctx, domain.AvatarEntityTeam, teamID, file,
		func(ctx context.Context, path string) (*string, error) {
			return uc.updateTeamAvatarURL(ctx, teamID, path, func(team *domain.Team) error {
				if team.CaptainID != callerID {
					return apperr.ErrNotTeamCaptain
				}

				if team.IsBanned {
					return apperr.ErrTeamBanned
				}

				return nil
			})
		},
		func(ctx context.Context) { uc.invalidateCache(ctx, nil, &teamID) },
	)
}

func (uc *AvatarUseCase) DeleteTeamAvatar(ctx context.Context, teamID, callerID uuid.UUID) error {
	oldAvatarURL, err := uc.clearTeamAvatarURL(ctx, teamID, func(team *domain.Team) error {
		if team.CaptainID != callerID {
			return apperr.ErrNotTeamCaptain
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("AvatarUseCase - DeleteTeamAvatar - clearTeamAvatarURL: %w", err)
	}

	if oldAvatarURL != nil && *oldAvatarURL != "" {
		uc.goDeleteOldAvatar(ctx, *oldAvatarURL, domain.ThumbPathFromFull(*oldAvatarURL))
	}

	uc.invalidateCache(ctx, nil, &teamID)

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

	return uc.resolvePresignedURLs(ctx, cacheutil.KeyAvatarTeam(teamID.String()), team.AvatarURL)
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
			_, thumbURL, resolveErr := uc.resolvePresignedURLs(gctx, cacheutil.KeyAvatarTeam(teamID.String()), team.AvatarURL)
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

// GetTeamAvatarStoragePathBatch returns the raw storage paths for a batch of
// teams without generating presigned URLs. Used by the scoreboard so that the
// frontend can build same-origin proxy URLs (/api/v1/avatars/<path>).
func (uc *AvatarUseCase) GetTeamAvatarStoragePathBatch(ctx context.Context, teamIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(teamIDs) == 0 {
		return nil, nil
	}

	teams, err := uc.deps.TeamRepo.GetByIDs(ctx, teamIDs)
	if err != nil {
		return nil, fmt.Errorf("AvatarUseCase - GetTeamAvatarStoragePathBatch - GetByIDs: %w", err)
	}

	result := make(map[uuid.UUID]string, len(teams))
	for teamID, team := range teams {
		if team.AvatarURL != nil && *team.AvatarURL != "" {
			result[teamID] = *team.AvatarURL
		}
	}

	return result, nil
}
