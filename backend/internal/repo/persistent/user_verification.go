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

func (r *UserRepo) SetVerified(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).UpdateUserVerified(ctx, sqlc.UpdateUserVerifiedParams{ID: userID, IsVerified: true, VerifiedAt: pgutil.TimeToTimestamptz(&now)})
	}, apperr.ErrUserNotFound, "UserRepo - SetVerified")

	return err
}

func (r *UserRepo) SetUnverified(ctx context.Context, userID uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).UpdateUserVerified(ctx, sqlc.UpdateUserVerifiedParams{ID: userID, IsVerified: false, VerifiedAt: pgutil.TimeToTimestamptz(nil)})
	}, apperr.ErrUserNotFound, "UserRepo - SetUnverified")

	return err
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	rows, err := r.Q(ctx).UpdatePassword(ctx, sqlc.UpdatePasswordParams{
		ID:           userID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - UpdatePassword: %w", err)
	}

	if rows == 0 {
		return apperr.ErrUserNotFound
	}

	return nil
}
