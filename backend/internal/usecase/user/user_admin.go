package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func (uc *UserUseCase) AdminCreate(ctx context.Context, username, email, password, role string) (*domain.User, error) {
	email = normalizeEmail(email)

	uc.bcryptSem <- struct{}{}

	defer func() { <-uc.bcryptSem }()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - AdminCreate - GenerateFromPassword: %w", err)
	}

	if role == "" {
		role = string(domain.RoleUser)
	}

	if domain.Role(role) != domain.RoleUser && domain.Role(role) != domain.RoleAdmin {
		return nil, httperr.NewValidationErrorf("invalid role %q: must be one of [user, admin]", role)
	}

	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.Role(role),
	}

	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.registerCheckUniqueness(ctx, username, email); err != nil {
			return fmt.Errorf("UserUseCase - AdminCreate - registerCheckUniqueness: %w", err)
		}

		if err := uc.deps.UserRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("UserUseCase - AdminCreate - UserRepo.Create: %w", err)
		}

		if uc.deps.CompRepo != nil && uc.deps.SoloTeamCreator != nil {
			comp, err := uc.deps.CompRepo.Get(ctx)
			if err != nil {
				return fmt.Errorf("UserUseCase - AdminCreate - CompRepo.Get: %w", err)
			}

			if comp.Mode == domain.ModeSoloOnly {
				team, err := uc.deps.SoloTeamCreator.CreateSoloTeamForNewUser(ctx, user.ID)
				if err != nil {
					return fmt.Errorf("UserUseCase - AdminCreate - SoloTeamCreator.CreateSoloTeamForNewUser: %w", err)
				}

				user.TeamID = &team.ID
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - AdminCreate - TM.Run: %w", err)
	}

	return user, nil
}

func (uc *UserUseCase) AdminUpdate(ctx context.Context, userID uuid.UUID, username, email, role, password *string, isVerified *bool) (*domain.User, error) {
	if email != nil {
		norm := normalizeEmail(*email)
		email = &norm
	}

	if role != nil && domain.Role(*role) != domain.RoleUser && domain.Role(*role) != domain.RoleAdmin {
		return nil, httperr.NewValidationErrorf("invalid role %q: must be one of [user, admin]", *role)
	}

	var passwordHash *string

	if password != nil {
		uc.bcryptSem <- struct{}{}

		defer func() { <-uc.bcryptSem }()

		hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("UserUseCase - AdminUpdate - GenerateFromPassword: %w", err)
		}

		h := string(hash)
		passwordHash = &h
	}

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			return fmt.Errorf("UserUseCase - AdminUpdate - UserRepo.Lock: %w", err)
		}

		current, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("UserUseCase - AdminUpdate - UserRepo.GetByID: %w", err)
		}

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

	if uc.deps.UserCache != nil {
		uc.deps.UserCache.InvalidateUser(context.WithoutCancel(ctx), userID)
	}

	return user, nil
}

func (uc *UserUseCase) AdminDelete(ctx context.Context, userID, actorID uuid.UUID) error {
	if userID == actorID {
		return httperr.ErrAccessDenied
	}

	var scoreboardInvalidateTeamID *uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			if errors.Is(err, httperr.ErrUserNotFound) {
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
			return httperr.ErrAccessDenied
		}

		if u.TeamID != nil && uc.deps.TeamRepo != nil {
			if err := uc.deps.TeamRepo.Lock(ctx, *u.TeamID); err != nil {
				return fmt.Errorf("UserUseCase - AdminDelete - TeamRepo.Lock: %w", err)
			}

			team, err := uc.deps.TeamRepo.GetByID(ctx, *u.TeamID)
			if err == nil && team != nil {
				if team.CaptainID == userID {
					return httperr.ErrCaptainCannotBeDeleted
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

	postCtx := context.WithoutCancel(ctx)

	if uc.deps.JWTService != nil {
		if err := uc.deps.JWTService.RevokeAllForUser(postCtx, userID); err != nil && uc.deps.Logger != nil {
			uc.deps.Logger.WithError(err).Warn("UserUseCase - AdminDelete - RevokeAllForUser")
		}
	}

	if uc.deps.UserCache != nil {
		uc.deps.UserCache.InvalidateUser(postCtx, userID)
	}

	if scoreboardInvalidateTeamID != nil && uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(postCtx, *scoreboardInvalidateTeamID)
	}

	if scoreboardInvalidateTeamID != nil && uc.deps.TeamCache != nil {
		if err := uc.deps.TeamCache.Del(postCtx, cache.KeyTeam(scoreboardInvalidateTeamID.String())); err != nil && uc.deps.Logger != nil {
			uc.deps.Logger.WithError(err).Warn("UserUseCase - AdminDelete - TeamCache.Del")
		}
	}

	if scoreboardInvalidateTeamID != nil && uc.deps.ChallengeListCache != nil {
		uc.deps.ChallengeListCache.InvalidateAll(postCtx)
	}

	return nil
}
