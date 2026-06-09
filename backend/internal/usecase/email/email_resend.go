package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/emailaddr"
)

// ResendVerificationByEmail looks up a user by email address and resends the
// verification email if the account exists and is not yet verified. To prevent
// user enumeration the function always returns nil regardless of whether the
// email address is registered.
func (uc *EmailUseCase) ResendVerificationByEmail(ctx context.Context, email string) error {
	if !uc.deps.Enabled {
		return nil
	}

	email = emailaddr.Normalize(email)

	user, err := uc.deps.UserRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperr.ErrUserNotFound) {
			return nil
		}

		return fmt.Errorf("EmailUseCase - ResendVerificationByEmail - UserRepo.GetByEmail: %w", err)
	}

	if user.IsVerified {
		return nil
	}

	return uc.SendVerificationEmail(ctx, user)
}

func (uc *EmailUseCase) ResendVerification(ctx context.Context, userID uuid.UUID) error {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("EmailUseCase - ResendVerification - UserRepo.GetByID: %w", err)
	}

	if user.IsVerified {
		return nil
	}

	return uc.SendVerificationEmail(ctx, user)
}
