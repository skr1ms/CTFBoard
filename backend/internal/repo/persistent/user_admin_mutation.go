package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *UserRepo) UpdateAdmin(ctx context.Context, userID uuid.UUID, username, email, role, passwordHash *string, isVerified *bool) error {
	rows, err := r.Q(ctx).UpdateUserAdmin(ctx, sqlc.UpdateUserAdminParams{
		ID:           userID,
		Username:     username,
		Email:        email,
		Role:         role,
		IsVerified:   isVerified,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateAdmin: %w", err)
	}

	if rows == 0 {
		return apperr.ErrUserNotFound
	}

	return nil
}

func (r *UserRepo) UpdateProfile(ctx context.Context, userID uuid.UUID, username, email, passwordHash *string) error {
	rows, err := r.Q(ctx).UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		ID:           userID,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return apperr.ErrUserAlreadyExists
		}

		return fmt.Errorf("UserRepo - UpdateProfile: %w", err)
	}

	if rows == 0 {
		return apperr.ErrUserNotFound
	}

	return nil
}

func (r *UserRepo) Delete(ctx context.Context, userID uuid.UUID) error {
	err := r.Q(ctx).DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("UserRepo - Delete: %w", err)
	}

	return nil
}

func (r *UserRepo) Lock(ctx context.Context, userID uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) { return r.Q(ctx).LockUser(ctx, userID) }, apperr.ErrUserNotFound, "UserRepo - Lock")

	return err
}

func (r *UserRepo) Ban(ctx context.Context, userID uuid.UUID, reason string) error {
	bannedAt := time.Now()
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).BanUser(ctx, sqlc.BanUserParams{ID: userID, BannedAt: pgutil.TimeToTimestamptz(&bannedAt), BannedReason: &reason})
	}, apperr.ErrUserNotFound, "UserRepo - Ban")

	return err
}

func (r *UserRepo) Unban(ctx context.Context, userID uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) { return r.Q(ctx).UnbanUser(ctx, userID) }, apperr.ErrUserNotFound, "UserRepo - Unban")

	return err
}

func (r *UserRepo) SetWasInBannedTeamByIDs(ctx context.Context, userIDs []uuid.UUID, value bool) error {
	if len(userIDs) == 0 {
		return nil
	}

	err := r.Q(ctx).SetWasInBannedTeamByIDs(ctx, sqlc.SetWasInBannedTeamByIDsParams{
		WasInBannedTeam: value,
		Column2:         userIDs,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - SetWasInBannedTeamByIDs: %w", err)
	}

	return nil
}
