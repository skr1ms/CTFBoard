package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	if u.Role == "" {
		u.Role = domain.RoleUser
	}

	EnsureID(&u.ID)
	u.CreatedAt = time.Now()

	err := r.Q(ctx).CreateUser(ctx, sqlc.CreateUserParams{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
		IsVerified:   u.IsVerified,
		CreatedAt:    pgutil.TimeToTimestamptz(&u.CreatedAt),
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return apperr.ErrUserAlreadyExists
		}

		return fmt.Errorf("UserRepo - Create: %w", err)
	}

	return nil
}
