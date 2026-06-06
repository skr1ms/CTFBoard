package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

func (uc *UserUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.User, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetByID - UserRepo.GetByID: %w", err)
	}

	return user, nil
}

// GetProfile fetches user, solves, and competition settings in parallel via
// errgroup, then filters the solve list to only include entries at or before
// the freeze time when the competition is currently frozen.
func (uc *UserUseCase) GetProfile(ctx context.Context, userID uuid.UUID) (*usecase.UserProfile, error) {
	v, err, _ := uc.profileSF.Do(userID.String(), func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		var (
			user   *domain.User
			solves []*domain.Solve
			comp   *domain.Competition
		)

		g, gCtx := errgroup.WithContext(loadCtx)
		g.Go(func() error {
			var err error

			user, err = uc.deps.UserRepo.GetByID(gCtx, userID)
			if err != nil {
				return fmt.Errorf("UserUseCase - GetProfile - UserRepo.GetByID: %w", err)
			}

			return nil
		})
		g.Go(func() error {
			var err error

			solves, err = uc.deps.SolveRepo.GetByUserID(gCtx, userID)
			if err != nil {
				return fmt.Errorf("UserUseCase - GetProfile - SolveRepo.GetByUserID: %w", err)
			}

			return nil
		})
		g.Go(func() error {
			if uc.deps.CompRepo == nil {
				return nil
			}

			var err error

			comp, err = uc.deps.CompRepo.Get(gCtx)
			if err != nil {
				return fmt.Errorf("UserUseCase - GetProfile - CompRepo.Get: %w", err)
			}

			return nil
		})

		if err := g.Wait(); err != nil {
			return nil, fmt.Errorf("UserUseCase - GetProfile - errgroup.Wait: %w", err)
		}

		if comp != nil && comp.IsFreezeActive() && comp.FreezeTime != nil {
			freezeAt := *comp.FreezeTime
			filtered := solves[:0]

			for _, s := range solves {
				if !s.SolvedAt.After(freezeAt) {
					filtered = append(filtered, s)
				}
			}

			solves = filtered
		}

		user.PasswordHash = ""

		return &usecase.UserProfile{
			User:   user,
			Solves: solves,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	return v.(*usecase.UserProfile), nil
}

// profileCheckUniqueness verifies that the requested username and email are not
// already taken by another user. currentUsername and currentEmail are the
// caller's existing values; fields that did not change are skipped to avoid
// spurious conflicts.
func (uc *UserUseCase) profileCheckUniqueness(ctx context.Context, currentUsername, currentEmail string, username, email *string) error {
	if username != nil && *username != currentUsername {
		existing, err := uc.deps.UserRepo.GetByUsername(ctx, *username)
		if err == nil && existing != nil {
			return apperr.ErrUserAlreadyExists
		}

		if err != nil && !errors.Is(err, apperr.ErrUserNotFound) {
			return fmt.Errorf("UserUseCase - profileCheckUniqueness - UserRepo.GetByUsername: %w", err)
		}
	}

	if email != nil && *email != currentEmail {
		existing, err := uc.deps.UserRepo.GetByEmail(ctx, *email)
		if err == nil && existing != nil {
			return apperr.ErrUserAlreadyExists
		}

		if err != nil && !errors.Is(err, apperr.ErrUserNotFound) {
			return fmt.Errorf("UserUseCase - profileCheckUniqueness - UserRepo.GetByEmail: %w", err)
		}
	}

	return nil
}

// UpdateProfile updates username, email, and/or password for the authenticated
// user. When email or password is being changed, the caller must supply the
// current password for re-verification (unless the account has no password hash,
// i.e. OAuth-only). A new password hash is produced under the CPU semaphore
// Inside a transaction the user row is locked, uniqueness is rechecked against
// the latest state to prevent races, and the profile is updated; an email change
// additionally marks the account as unverified. After the transaction commits a
// verification email is sent on a best-effort basis, and a password change
// triggers revocation of all existing JWTs except the current session so that
// other devices are signed out.
func (uc *UserUseCase) UpdateProfile(ctx context.Context, userID uuid.UUID, username, email, currentPassword, newPassword *string) (*domain.User, error) {
	if email != nil {
		norm := normalizeEmail(*email)
		email = &norm
	}

	current, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - UpdateProfile - UserRepo.GetByID: %w", err)
	}

	if (newPassword != nil || email != nil) && current.PasswordHash != "" {
		if currentPassword == nil {
			return nil, apperr.ErrInvalidCredentials
		}

		err := bcrypt.CompareHashAndPassword([]byte(current.PasswordHash), []byte(*currentPassword))
		if err != nil {
			return nil, apperr.ErrInvalidCredentials
		}
	}

	var passwordHash *string

	if newPassword != nil {
		uc.bcryptSem <- struct{}{}

		defer func() { <-uc.bcryptSem }()

		hash, err := bcrypt.GenerateFromPassword([]byte(*newPassword), uc.bcryptCost())
		if err != nil {
			return nil, fmt.Errorf("UserUseCase - UpdateProfile - GenerateFromPassword: %w", err)
		}

		h := string(hash)
		passwordHash = &h
	}

	var emailChanged bool

	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			return fmt.Errorf("UserUseCase - UpdateProfile - UserRepo.Lock: %w", err)
		}
		// Re-fetch inside the transaction so uniqueness checks are consistent
		fresh, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("UserUseCase - UpdateProfile - UserRepo.GetByID (tx): %w", err)
		}

		if err := uc.profileCheckUniqueness(ctx, fresh.Username, fresh.Email, username, email); err != nil {
			return fmt.Errorf("UserUseCase - UpdateProfile - profileCheckUniqueness: %w", err)
		}

		if err := uc.deps.UserRepo.UpdateProfile(ctx, userID, username, email, passwordHash); err != nil {
			return fmt.Errorf("UserUseCase - UpdateProfile - UserRepo.UpdateProfile: %w", err)
		}

		emailChanged = email != nil && *email != fresh.Email
		if emailChanged {
			err := uc.deps.UserRepo.SetUnverified(ctx, userID)
			if err != nil {
				return fmt.Errorf("UserUseCase - UpdateProfile - UserRepo.SetUnverified: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - UpdateProfile - TM.Run: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - UpdateProfile - UserRepo.GetByID: %w", err)
	}

	if emailChanged && uc.deps.EmailSender != nil {
		_ = uc.deps.EmailSender.SendVerificationEmail(ctx, user)
	}

	if newPassword != nil {
		err := uc.deps.JWTService.RevokeAllForUser(context.WithoutCancel(ctx), userID)
		if err != nil {
			uc.deps.Logger.WithError(err).Error("UserUseCase - UpdateProfile - RevokeAllForUser")
		}
	}

	return user, nil
}
