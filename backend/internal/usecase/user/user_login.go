package user

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// Login authenticates a user by email and password. It normalizes the email,
// checks whether the address is locked out due to previous failed attempts, and
// looks up the user. When the user is not found it still executes a dummy bcrypt
// comparison under the CPU semaphore so that the response time does not leak
// whether the address exists (timing-attack mitigation). Users marked
// Directly banned users and former members of banned teams can still log in so
// the frontend can fetch /auth/me and render ban/appeal state. Protected CTF
// actions remain blocked by middleware and usecase guards.
// An OAuth-only sentinel password is checked before the real hash comparison.
// On success the failed-login counter is cleared and a JWT token pair is issued.
func (uc *UserUseCase) Login(ctx context.Context, email, password string) (*usecase.TokenPair, error) {
	email = normalizeEmail(email)

	if uc.deps.FailedLogin != nil {
		locked, err := uc.deps.FailedLogin.IsLocked(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("UserUseCase - Login - FailedLogin.IsLocked: %w", err)
		}

		if locked {
			return nil, apperr.ErrTooManyRequests
		}
	}

	user, err := uc.deps.UserRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperr.ErrUserNotFound) {
			func() {
				uc.bcryptSem <- struct{}{}

				defer func() { <-uc.bcryptSem }()

				_ = bcrypt.CompareHashAndPassword(uc.dummyHash, []byte(password))
			}()
			uc.recordFailedLogin(ctx, email)

			return nil, apperr.ErrInvalidCredentials
		}

		return nil, fmt.Errorf("UserUseCase - Login - UserRepo.GetByEmail: %w", err)
	}

	if !user.HasLocalPassword() {
		uc.recordFailedLogin(ctx, email)

		return nil, apperr.ErrInvalidCredentials
	}

	uc.bcryptSem <- struct{}{}

	defer func() { <-uc.bcryptSem }()

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		uc.recordFailedLogin(ctx, email)

		return nil, apperr.ErrInvalidCredentials
	}

	uc.clearFailedLogin(ctx, email)

	return issueUserTokenPair(ctx, uc.deps.JWTService, user, "UserUseCase - Login")
}

func (uc *UserUseCase) recordFailedLogin(ctx context.Context, email string) {
	if uc.deps.FailedLogin != nil {
		_ = uc.deps.FailedLogin.RecordFailed(ctx, email)
	}
}

func (uc *UserUseCase) clearFailedLogin(ctx context.Context, email string) {
	if uc.deps.FailedLogin != nil {
		_ = uc.deps.FailedLogin.ClearFailed(ctx, email)
	}
}
