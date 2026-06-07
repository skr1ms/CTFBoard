package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const (
	registrationVisibilityPublic  = "public"
	registrationVisibilityPrivate = "private"
	defaultPasswordMinLength      = 8
	maxUsersLockKey               = int64(0x4354467573657273)
)

type registrationPolicyMode int

const (
	registrationPolicyLocal registrationPolicyMode = iota
	registrationPolicyOAuth
)

type registrationPolicyDeps struct {
	UserRepo    repo.UserRepository
	CompParamUC usecase.CompetitionParamUseCase
}

func enforceRegistrationPolicy(ctx context.Context, deps registrationPolicyDeps, settings *domain.Settings, mode registrationPolicyMode, registrationCode string) error {
	if settings != nil && !settings.RegistrationOpen {
		return apperr.ErrRegistrationClosed
	}

	visibility := registrationVisibilityPublic

	if deps.CompParamUC != nil {
		visibility = strings.TrimSpace(deps.CompParamUC.GetString(ctx, "registration_visibility", registrationVisibilityPublic))
	}

	if visibility == "" {
		visibility = registrationVisibilityPublic
	}

	if visibility != registrationVisibilityPublic {
		return apperr.ErrVisibilityForbidden
	}

	configuredCode := ""

	if deps.CompParamUC != nil {
		configuredCode = strings.TrimSpace(deps.CompParamUC.GetString(ctx, "registration_code", ""))
	}

	if configuredCode != "" {
		if mode == registrationPolicyOAuth {
			return apperr.ErrRegistrationCodeRequired
		}

		code := strings.TrimSpace(registrationCode)
		if code == "" {
			return apperr.ErrRegistrationCodeRequired
		}

		if !strings.EqualFold(code, configuredCode) {
			return apperr.ErrInvalidRegistrationCode
		}
	}

	if settings != nil && settings.MaxUsers > 0 {
		if err := deps.UserRepo.AcquireAdvisoryLock(ctx, maxUsersLockKey); err != nil {
			return fmt.Errorf("UserUseCase - enforceRegistrationPolicy - AcquireAdvisoryLock: %w", err)
		}

		currentCount, err := deps.UserRepo.CountActiveUsers(ctx)
		if err != nil {
			return fmt.Errorf("UserUseCase - enforceRegistrationPolicy - CountActiveUsers: %w", err)
		}

		if currentCount >= int64(settings.MaxUsers) {
			return apperr.ErrMaxUsersReached
		}
	}

	return nil
}

func (uc *UserUseCase) validateConfiguredPasswordLength(ctx context.Context, password string) error {
	if uc.deps.CompParamUC == nil {
		return nil
	}

	minLen := uc.deps.CompParamUC.GetInt(ctx, "password_min_length", defaultPasswordMinLength)
	if minLen <= 0 {
		return nil
	}

	if len(password) < minLen {
		return apperr.NewValidationErrorf("password must be at least %d characters", minLen)
	}

	return nil
}

func acquireUserRegistrationLocks(ctx context.Context, userRepo repo.UserRepository, username, email string) error {
	return repo.AcquireRegistrationAdvisoryLocks(ctx, userRepo,
		repo.RegistrationAdvisoryLock{Label: "email", Scope: repo.RegistrationLockEmail, Value: email},
		repo.RegistrationAdvisoryLock{Label: "username", Scope: repo.RegistrationLockUsername, Value: username},
	)
}
