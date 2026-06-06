package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/ctxutil"
)

// AdminCreate creates a new user account with an optional role.
// bcrypt hashing runs before the transaction and is bounded by bcryptSem to
// limit concurrent CPU-intensive hashes. Inside the transaction, uniqueness of
// username and email is verified before insert. In solo_only competition mode
// a personal team is automatically created for the new user via SoloTeamCreator.
func (uc *UserUseCase) AdminCreate(ctx context.Context, username, email, password, role string) (*domain.User, error) {
	email = normalizeEmail(email)

	uc.bcryptSem <- struct{}{}

	defer func() { <-uc.bcryptSem }()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), uc.bcryptCost())
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - AdminCreate - GenerateFromPassword: %w", err)
	}

	if role == "" {
		role = string(domain.RoleUser)
	}

	if domain.Role(role) != domain.RoleUser && domain.Role(role) != domain.RoleAdmin {
		return nil, apperr.NewValidationErrorf("invalid role %q: must be one of [user, admin]", role)
	}

	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.Role(role),
		IsVerified:   true,
	}

	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := repo.AcquireRegistrationAdvisoryLocks(ctx, uc.deps.UserRepo,
			repo.RegistrationAdvisoryLock{Label: "email", Scope: repo.RegistrationLockEmail, Value: email},
			repo.RegistrationAdvisoryLock{Label: "username", Scope: repo.RegistrationLockUsername, Value: username},
		); err != nil {
			return fmt.Errorf("UserUseCase - AdminCreate - %w", err)
		}

		if err := uc.registerCheckUniqueness(ctx, username, email); err != nil {
			return fmt.Errorf("UserUseCase - AdminCreate - registerCheckUniqueness: %w", err)
		}

		if err := uc.deps.UserRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("UserUseCase - AdminCreate - UserRepo.Create: %w", err)
		}

		if err := ensureSoloTeamIfRequired(ctx, uc.deps.CompRepo, uc.deps.SoloTeamCreator, user); err != nil {
			return fmt.Errorf("UserUseCase - AdminCreate - ensureSoloTeamIfRequired: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - AdminCreate - TM.Run: %w", err)
	}

	return user, nil
}

// AdminUpdate modifies an existing user account. If a new password is supplied,
// bcrypt hashing runs under bcryptSem before the transaction. Inside the
// transaction, UserRepo.Lock acquires a pessimistic row lock, the current record
// is fetched for TOCTOU-safe uniqueness rechecking, and UpdateAdmin applies the
// partial update. After commit, the user cache entry is invalidated.
func (uc *UserUseCase) AdminUpdate(ctx context.Context, userID uuid.UUID, username, email, role, password *string, isVerified *bool) (*domain.User, error) {
	if email != nil {
		norm := normalizeEmail(*email)
		email = &norm
	}

	if role != nil && domain.Role(*role) != domain.RoleUser && domain.Role(*role) != domain.RoleAdmin {
		return nil, apperr.NewValidationErrorf("invalid role %q: must be one of [user, admin]", *role)
	}

	var passwordHash *string

	if password != nil {
		uc.bcryptSem <- struct{}{}

		defer func() { <-uc.bcryptSem }()

		hash, err := bcrypt.GenerateFromPassword([]byte(*password), uc.bcryptCost())
		if err != nil {
			return nil, fmt.Errorf("UserUseCase - AdminUpdate - GenerateFromPassword: %w", err)
		}

		h := string(hash)
		passwordHash = &h
	}

	var previousRole domain.Role

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			return fmt.Errorf("UserUseCase - AdminUpdate - UserRepo.Lock: %w", err)
		}

		current, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("UserUseCase - AdminUpdate - UserRepo.GetByID: %w", err)
		}

		previousRole = current.Role

		if err := uc.profileCheckUniqueness(ctx, current.Username, current.Email, username, email); err != nil {
			return fmt.Errorf("UserUseCase - AdminUpdate - profileCheckUniqueness: %w", err)
		}

		if err := uc.deps.UserRepo.UpdateAdmin(ctx, userID, username, email, role, passwordHash, isVerified); err != nil {
			return fmt.Errorf("UserUseCase - AdminUpdate - UserRepo.UpdateAdmin: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - AdminUpdate - TM.Run: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - AdminUpdate - UserRepo.GetByID: %w", err)
	}

	postCtx, postCancel := ctxutil.PostCommitContext(ctx)
	defer postCancel()

	// Revoke JWT when password changed or role demoted (admin -> user)
	needsRevoke := passwordHash != nil

	if role != nil && previousRole == domain.RoleAdmin && domain.Role(*role) == domain.RoleUser {
		needsRevoke = true
	}

	if needsRevoke && uc.deps.JWTService != nil {
		if err := uc.deps.JWTService.RevokeAllForUser(postCtx, userID); err != nil {
			uc.deps.Logger.WithError(err).Warn("UserUseCase - AdminUpdate - RevokeAllForUser")
		}
	}

	cacheutil.InvalidateUser(postCtx, uc.deps.UserCache, userID)

	return user, nil
}

// AdminDelete permanently removes a user account with cascading cleanup.
// Guards enforced before the transaction: self-delete is rejected. Inside the
// transaction: UserRepo.Lock + admin-protect check + captain-guard (captains
// cannot be deleted while leading a team). banUserRemoveSolvesAndAdjustScores
// strips solves and recalculates team scores; custom field values are deleted.
// After the transaction: JWT tokens are revoked and user/scoreboard/team/challenge
// caches are invalidated with a bounded post-commit context so cleanup survives
// request cancellation without becoming unbounded.
func (uc *UserUseCase) AdminDelete(ctx context.Context, userID, actorID uuid.UUID) error {
	if userID == actorID {
		return apperr.ErrAccessDenied
	}

	var scoreboardInvalidateTeamID *uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			if errors.Is(err, apperr.ErrUserNotFound) {
				return nil
			}

			return fmt.Errorf("UserUseCase - AdminDelete - UserRepo.Lock: %w", err)
		}

		u, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("UserUseCase - AdminDelete - UserRepo.GetByID: %w", err)
		}

		if u == nil {
			return nil
		}

		if u.Role == domain.RoleAdmin {
			return apperr.ErrAccessDenied
		}

		if u.TeamID != nil && uc.deps.TeamRepo != nil {
			if err := uc.deps.TeamRepo.Lock(ctx, *u.TeamID); err != nil {
				return fmt.Errorf("UserUseCase - AdminDelete - TeamRepo.Lock: %w", err)
			}

			team, err := uc.deps.TeamRepo.GetByID(ctx, *u.TeamID)
			if err == nil && team != nil {
				if team.CaptainID == userID {
					return apperr.ErrCaptainCannotBeDeleted
				}

				if err := uc.banUserRemoveSolvesAndAdjustScores(ctx, team.ID, userID); err != nil {
					return fmt.Errorf("UserUseCase - AdminDelete - banUserRemoveSolvesAndAdjustScores: %w", err)
				}

				scoreboardInvalidateTeamID = &team.ID
			}
		}

		if uc.deps.FieldValueRepo != nil {
			if err := uc.deps.FieldValueRepo.DeleteByEntityID(ctx, userID); err != nil {
				return fmt.Errorf("UserUseCase - AdminDelete - FieldValueRepo.DeleteByEntityID: %w", err)
			}
		}

		return uc.deps.UserRepo.Delete(ctx, userID)
	}); err != nil {
		return fmt.Errorf("UserUseCase - AdminDelete - TM.Run: %w", err)
	}

	postCtx, postCancel := ctxutil.PostCommitContext(ctx)
	defer postCancel()

	if uc.deps.JWTService != nil {
		if err := uc.deps.JWTService.RevokeAllForUser(postCtx, userID); err != nil {
			uc.deps.Logger.WithError(err).Warn("UserUseCase - AdminDelete - RevokeAllForUser")
		}
	}

	cacheutil.InvalidateUser(postCtx, uc.deps.UserCache, userID)

	if scoreboardInvalidateTeamID != nil {
		cacheutil.InvalidateScoreboardForTeam(postCtx, uc.deps.ScoreboardCache, *scoreboardInvalidateTeamID)
		cacheutil.InvalidateTeam(postCtx, uc.deps.TeamCache, uc.deps.Logger, *scoreboardInvalidateTeamID)
		cacheutil.InvalidateChallengeList(postCtx, uc.deps.ChallengeListCache)
	}

	return nil
}
