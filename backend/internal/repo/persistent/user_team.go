package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *UserRepo) UpdateTeamIDBatch(ctx context.Context, userIDs []uuid.UUID, teamID *uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	err := r.Q(ctx).UpdateUserTeamIDBatch(ctx, sqlc.UpdateUserTeamIDBatchParams{
		TeamID:  teamID,
		Column2: userIDs,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateTeamIDBatch: %w", err)
	}

	return nil
}

func (r *UserRepo) FilterIDsByTeamIDNull(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	ids, err := r.Q(ctx).ListUserIDsWithTeamIDNull(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - FilterIDsByTeamIDNull: %w", err)
	}

	return ids, nil
}

func (r *UserRepo) FilterIDsByTeamIDNullAndNotBanned(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	ids, err := r.Q(ctx).ListUserIDsWithTeamIDNullAndNotBanned(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - FilterIDsByTeamIDNullAndNotBanned: %w", err)
	}

	return ids, nil
}

func (r *UserRepo) UpdateTeamID(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).UpdateUserTeamID(ctx, sqlc.UpdateUserTeamIDParams{ID: userID, TeamID: teamID})
	}, apperr.ErrUserNotFound, "UserRepo - UpdateTeamID")

	return err
}
