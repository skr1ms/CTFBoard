package user

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"slices"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	validation "github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// registrationAdvisoryKey derives a PostgreSQL advisory lock key from a prefix+value
// pair using FNV-1a 64-bit hash, capped to int64 range. Used to serialize concurrent
// registrations for the same username or email and prevent TOCTOU races.
func registrationAdvisoryKey(prefix, value string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(prefix))
	_, _ = h.Write([]byte(value))

	u := min(h.Sum64(), 1<<63-1)

	return int64(u)
}

func acquireRegistrationAdvisoryLocks(ctx context.Context, locker registrationAdvisoryLocker, op string, locks ...registrationAdvisoryLock) error {
	slices.SortFunc(locks, func(a, b registrationAdvisoryLock) int {
		if a.key < b.key {
			return -1
		}

		if a.key > b.key {
			return 1
		}

		return 0
	})

	var lastKey int64

	for i, lock := range locks {
		if i > 0 && lock.key == lastKey {
			continue
		}

		if err := locker.AcquireAdvisoryLock(ctx, lock.key); err != nil {
			return fmt.Errorf("%s - AcquireAdvisoryLock(%s): %w", op, lock.label, err)
		}

		lastKey = lock.key
	}

	return nil
}

// Register creates a new user account. It normalizes the email address, enforces
// custom-field limits and validates field values against the configured schema, then
// hashes the password under a semaphore to cap concurrent bcrypt work. Inside a
// transaction it acquires two PostgreSQL advisory locks (one per unique key) in a
// deterministic order to prevent TOCTOU races between the uniqueness check and the
// INSERT. After the user row is created it persists any custom field values and, when
// the competition is in solo_only mode, auto-creates a solo team within the same
// transaction so a rollback on failure leaves no orphaned rows. A verification email
// is dispatched outside the transaction on a best-effort basis.
func (uc *UserUseCase) Register(ctx context.Context, username, email, password string, customFields map[string]string) (*domain.User, error) {
	email = normalizeEmail(email)

	if err := validation.ValidateCustomFieldEnvelope(customFields); err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - ValidateCustomFieldEnvelope: %w", apperr.NewValidationErrorf("%v", err))
	}

	customFields = validation.SanitizeCustomFieldValues(customFields)
	if err := uc.registerValidateCustomFields(ctx, customFields); err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - registerValidateCustomFields: %w", err)
	}

	uc.bcryptSem <- struct{}{}

	defer func() { <-uc.bcryptSem }()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), uc.bcryptCost())
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - GenerateFromPassword: %w", err)
	}

	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
	}

	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if uc.deps.SettingsRepo != nil {
			settings, err := uc.deps.SettingsRepo.Get(ctx)
			if err != nil {
				return fmt.Errorf("UserUseCase - Register - SettingsRepo.Get: %w", err)
			}

			if !settings.RegistrationOpen {
				return apperr.ErrRegistrationClosed
			}
		}

		err := acquireRegistrationAdvisoryLocks(ctx, uc.deps.UserRepo, "UserUseCase - Register",
			registrationAdvisoryLock{label: "email", key: registrationAdvisoryKey("reg:email:", email)},
			registrationAdvisoryLock{label: "username", key: registrationAdvisoryKey("reg:username:", username)},
		)
		if err != nil {
			return err
		}

		err = uc.registerCheckUniqueness(ctx, username, email)
		if err != nil {
			return fmt.Errorf("UserUseCase - Register - registerCheckUniqueness: %w", err)
		}

		err = uc.deps.UserRepo.Create(ctx, user)
		if err != nil {
			return fmt.Errorf("UserUseCase - Register - UserRepo.Create: %w", err)
		}

		if uc.deps.FieldValueRepo != nil && len(customFields) > 0 {
			err := uc.deps.FieldValueRepo.SetValues(ctx, user.ID, customFields)
			if err != nil {
				return fmt.Errorf("UserUseCase - Register - FieldValueRepo.SetValues: %w", err)
			}
		}
		// Solo team creation is inside the transaction so that a failure rolls
		// back the entire registration rather than leaving a user without a team
		// CreateSoloTeamForNewUser internally calls TM.Run which reuses this tx
		if err := ensureSoloTeamIfRequired(ctx, uc.deps.CompRepo, uc.deps.SoloTeamCreator, user); err != nil {
			return fmt.Errorf("UserUseCase - Register - ensureSoloTeamIfRequired: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - TM.Run: %w", err)
	}

	return user, nil
}

// registerValidateCustomFields parses the string-keyed custom field map into
// uuid.UUID-keyed values (returning ErrValidation on a malformed key) and then
// delegates value validation to FieldValidator. It still calls FieldValidator
// for an empty map so required fields are enforced.
func (uc *UserUseCase) registerValidateCustomFields(ctx context.Context, customFields map[string]string) error {
	if uc.deps.FieldValidator == nil {
		return nil
	}

	fieldValues := make(map[uuid.UUID]string, len(customFields))

	for k, v := range customFields {
		id, err := uuid.Parse(k)
		if err != nil {
			return apperr.NewValidationErrorf("invalid custom field key")
		}

		fieldValues[id] = v
	}

	err := uc.deps.FieldValidator.ValidateValues(ctx, domain.EntityTypeUser, fieldValues)
	if err != nil {
		return fmt.Errorf("UserUseCase - registerValidateCustomFields - FieldValidator.ValidateValues: %w", err)
	}

	return nil
}

// registerCheckUniqueness verifies that neither the username nor the email is already
// taken. Both checks run sequentially inside the registration advisory lock so the
// uniqueness window is as small as possible.
func (uc *UserUseCase) registerCheckUniqueness(ctx context.Context, username, email string) error {
	existing, err := uc.deps.UserRepo.GetByUsername(ctx, username)
	if err == nil && existing != nil {
		return fmt.Errorf("UserUseCase - registerCheckUniqueness - username taken: %w", apperr.ErrUserAlreadyExists)
	}

	if err != nil && !errors.Is(err, apperr.ErrUserNotFound) {
		return fmt.Errorf("UserUseCase - registerCheckUniqueness - UserRepo.GetByUsername: %w", err)
	}

	existing, err = uc.deps.UserRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return fmt.Errorf("UserUseCase - registerCheckUniqueness - email taken: %w", apperr.ErrUserAlreadyExists)
	}

	if err != nil && !errors.Is(err, apperr.ErrUserNotFound) {
		return fmt.Errorf("UserUseCase - registerCheckUniqueness - UserRepo.GetByEmail: %w", err)
	}

	return nil
}

// ensureSoloTeamIfRequired creates a solo team for user when the competition is in
// solo-only mode. No-ops when compRepo or creator are nil.
func ensureSoloTeamIfRequired(ctx context.Context, compRepo repo.CompetitionRepository, creator SoloTeamCreator, user *domain.User) error {
	if compRepo == nil || creator == nil {
		return nil
	}

	comp, err := compRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("ensureSoloTeamIfRequired - CompRepo.Get: %w", err)
	}

	if comp.Mode != domain.ModeSoloOnly {
		return nil
	}

	team, err := creator.CreateSoloTeamForNewUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("ensureSoloTeamIfRequired - CreateSoloTeamForNewUser: %w", err)
	}

	user.TeamID = &team.ID

	return nil
}
