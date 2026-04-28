package user

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// registrationAdvisoryKey derives a PostgreSQL advisory lock key from a prefix+value
// pair using FNV-1a 64-bit hash, capped to int64 range. Used to serialize concurrent
// registrations for the same username or email and prevent TOCTOU races.
func registrationAdvisoryKey(prefix, value string) int64 {
	h := fnv.New64a()
	h.Write([]byte(prefix)) //nolint:revive // hash.Write never returns error
	h.Write([]byte(value))  //nolint:revive // hash.Write never returns error

	u := min(h.Sum64(), 1<<63-1)

	return int64(u)
}

// FieldValidator validates custom field values (e.g. on registration)
// Implemented by *settings.FieldValidator.
type FieldValidator interface {
	ValidateValues(ctx context.Context, entityType domain.EntityType, values map[uuid.UUID]string) error
}

// EmailVerificationSender sends verification email (e.g. usecase.EmailUseCase)
// Optional; when set, used to send verification after email change in UpdateProfile.
type EmailVerificationSender interface {
	SendVerificationEmail(ctx context.Context, user *domain.User) error
}

// FailedLoginTracker counts failed logins per email for lockout. Optional.
type FailedLoginTracker interface {
	IsLocked(ctx context.Context, email string) (bool, error)
	RecordFailed(ctx context.Context, email string) error
	ClearFailed(ctx context.Context, email string) error
}

// SoloTeamCreator creates a solo team for a user. Used to auto-create solo teams on registration when competition mode is solo_only.
type SoloTeamCreator interface {
	CreateSoloTeamForNewUser(ctx context.Context, userID uuid.UUID) (*domain.Team, error)
}

// PersonalNotificationSender sends a personal notification to a user. Optional; used e.g. to notify team captain when team falls below MinTeamSize after a ban.
type PersonalNotificationSender interface {
	CreatePersonal(ctx context.Context, userID uuid.UUID, title, content string, notifType domain.NotificationType) (*domain.UserNotification, error)
}

type UserUseCase struct {
	deps      UserDeps
	bcryptSem chan struct{}      // limits concurrent bcrypt to avoid CPU saturation
	dummyHash []byte             // pre-computed hash used for constant-time login when user is not found
	profileSF singleflight.Group // deduplicates concurrent GetProfile calls for the same userID
}

type UserDeps struct {
	UserRepo                   repo.UserRepository
	TeamRepo                   repo.TeamRepository
	SolveRepo                  repo.SolveRepository
	ChallengeRepo              repo.ChallengeRepository
	SubmissionRepo             repo.SubmissionRepository
	AwardRepo                  repo.AwardRepository
	HintRepo                   repo.HintRepository
	TM                         repo.TransactionManager
	JWTService                 jwtkit.Service
	FieldValidator             FieldValidator
	FieldValueRepo             repo.FieldValueRepository
	SettingsRepo               repo.SettingsRepository
	EmailSender                EmailVerificationSender
	FailedLogin                FailedLoginTracker
	CompRepo                   repo.CompetitionRepository
	SoloTeamCreator            SoloTeamCreator
	Logger                     logkit.Logger
	UserCache                  cache.UserCacheInvalidator
	ScoreboardCache            cache.ScoreboardCacheInvalidator
	ChallengeListCache         cache.ChallengeListCacheInvalidator
	TeamCache                  *cachekit.Cache
	PersonalNotificationSender PersonalNotificationSender
	CompParamUC                usecase.CompetitionParamUseCase
	BcryptCost                 int // 0 = bcrypt.DefaultCost; set to bcrypt.MinCost in tests to avoid race-detector slowdown
}

var _ usecase.UserUseCase = (*UserUseCase)(nil)

// NewUserUseCase builds a UserUseCase: creates a bcrypt semaphore sized to NumCPU*2
// to cap concurrent bcrypt operations, and pre-computes a dummy hash used in Login
// for constant-time responses when the username does not exist (timing-attack mitigation).
func NewUserUseCase(deps UserDeps) *UserUseCase {
	n := max(runtime.NumCPU()*2, 2)

	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-timing-pad-ctf-platform"), deps.bcryptCost())
	if err != nil {
		deps.Logger.Warn("UserUseCase - dummy bcrypt hash failed, using fallback")

		dummy = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy") // nosemgrep: generic.secrets.security.detected-bcrypt-hash
	}

	return &UserUseCase{deps: deps, bcryptSem: make(chan struct{}, n), dummyHash: dummy}
}

const (
	maxCustomFields   = 10
	maxCustomFieldKey = 100
	maxCustomFieldVal = 500
)

func validateCustomFieldsLimits(fields map[string]string) error {
	if len(fields) > maxCustomFields {
		return apperr.NewValidationErrorf("too many custom fields (max %d)", maxCustomFields)
	}

	for k, v := range fields {
		if len(k) > maxCustomFieldKey || len(v) > maxCustomFieldVal {
			return apperr.NewValidationErrorf("custom field key or value too long")
		}
	}

	return nil
}

func sanitizeCustomFieldValue(s string) string {
	var b strings.Builder

	for _, r := range s {
		if r >= 32 && r != 127 {
			b.WriteRune(r)
		}
	}

	return strings.TrimSpace(b.String())
}

func sanitizeCustomFields(fields map[string]string) map[string]string {
	if len(fields) == 0 {
		return fields
	}

	out := make(map[string]string, len(fields))
	for k, v := range fields {
		out[k] = sanitizeCustomFieldValue(v)
	}

	return out
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

	if err := validateCustomFieldsLimits(customFields); err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - validateCustomFieldsLimits: %w", err)
	}

	customFields = sanitizeCustomFields(customFields)
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

		keyEmail := registrationAdvisoryKey("reg:email:", email)

		keyUsername := registrationAdvisoryKey("reg:username:", username)
		if keyEmail > keyUsername {
			keyEmail, keyUsername = keyUsername, keyEmail
		}

		err := uc.deps.UserRepo.AcquireAdvisoryLock(ctx, keyEmail)
		if err != nil {
			return fmt.Errorf("UserUseCase - Register - AcquireAdvisoryLock(email): %w", err)
		}

		if keyUsername != keyEmail {
			err := uc.deps.UserRepo.AcquireAdvisoryLock(ctx, keyUsername)
			if err != nil {
				return fmt.Errorf("UserUseCase - Register - AcquireAdvisoryLock(username): %w", err)
			}
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
// delegates value validation to FieldValidator. No-ops when customFields is
// empty or FieldValidator is not wired.
func (uc *UserUseCase) registerValidateCustomFields(ctx context.Context, customFields map[string]string) error {
	if len(customFields) == 0 || uc.deps.FieldValidator == nil {
		return nil
	}

	fieldValues := make(map[uuid.UUID]string)

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

// Login authenticates a user by email and password. It normalizes the email,
// checks whether the address is locked out due to previous failed attempts, and
// looks up the user. When the user is not found it still executes a dummy bcrypt
// comparison under the CPU semaphore so that the response time does not leak
// whether the address exists (timing-attack mitigation). Ban status and an
// OAuth-only sentinel password are checked before the real hash comparison
// On success the failed-login counter is cleared and a JWT token pair is issued.
func (uc *UserUseCase) Login(ctx context.Context, email, password string) (*jwtkit.TokenPair, error) {
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

				_ = bcrypt.CompareHashAndPassword(uc.dummyHash, []byte(password)) //nolint:errcheck // intentional dummy
			}()
			uc.recordFailedLogin(ctx, email)

			return nil, apperr.ErrInvalidCredentials
		}

		return nil, fmt.Errorf("UserUseCase - Login - UserRepo.GetByEmail: %w", err)
	}

	if user.WasInBannedTeam && user.Role != domain.RoleAdmin {
		uc.recordFailedLogin(ctx, email)

		return nil, apperr.ErrInvalidCredentials
	}

	if user.PasswordHash == "" || user.PasswordHash == domain.OAuthOnlyPasswordSentinel {
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

	tokenPair, err := uc.deps.JWTService.GenerateTokenPair(ctx, user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - Login - JWTService.GenerateTokenPair: %w", err)
	}

	return tokenPair, nil
}

func (uc *UserUseCase) recordFailedLogin(ctx context.Context, email string) {
	if uc.deps.FailedLogin != nil {
		_ = uc.deps.FailedLogin.RecordFailed(ctx, email) //nolint:errcheck // best-effort
	}
}

func (uc *UserUseCase) clearFailedLogin(ctx context.Context, email string) {
	if uc.deps.FailedLogin != nil {
		_ = uc.deps.FailedLogin.ClearFailed(ctx, email) //nolint:errcheck // best-effort
	}
}

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
		sfCtx := context.WithoutCancel(ctx)

		var (
			user   *domain.User
			solves []*domain.Solve
			comp   *domain.Competition
		)

		g, gCtx := errgroup.WithContext(sfCtx)
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

// ListUsers returns a paginated user list inside a read-only transaction.
// When field=="ip" the search targets tracked IP addresses; otherwise it
// performs a standard username/email search.
func (uc *UserUseCase) ListUsers(ctx context.Context, search *string, field string, page, perPage int) (*usecase.Paginated[*domain.User], error) {
	offset := (page - 1) * perPage

	var (
		users []*domain.User
		total int64
	)

	err := uc.deps.TM.ReadOnly(ctx, func(roCtx context.Context) error {
		if field == "ip" && search != nil && *search != "" {
			var err error

			users, err = uc.deps.UserRepo.SearchByIP(roCtx, *search, perPage, offset)
			if err != nil {
				return fmt.Errorf("UserUseCase - ListUsers - UserRepo.SearchByIP: %w", err)
			}

			total, err = uc.deps.UserRepo.CountSearchByIP(roCtx, *search)
			if err != nil {
				return fmt.Errorf("UserUseCase - ListUsers - UserRepo.CountSearchByIP: %w", err)
			}

			return nil
		}

		var err error

		users, err = uc.deps.UserRepo.Search(roCtx, search, perPage, offset)
		if err != nil {
			return fmt.Errorf("UserUseCase - ListUsers - UserRepo.Search: %w", err)
		}

		total, err = uc.deps.UserRepo.CountSearch(roCtx, search)
		if err != nil {
			return fmt.Errorf("UserUseCase - ListUsers - UserRepo.CountSearch: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - ListUsers - TM.ReadOnly: %w", err)
	}

	return usecase.NewPaginated(users, total, page, perPage), nil
}

func (uc *UserUseCase) GetUserSolves(ctx context.Context, userID uuid.UUID) ([]*domain.SolveWithDetails, error) {
	if _, err := uc.deps.UserRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserSolves - UserRepo.GetByID: %w", err)
	}

	solves, err := uc.deps.SolveRepo.GetByUserIDWithDetails(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserSolves - SolveRepo.GetByUserIDWithDetails: %w", err)
	}

	return scoring.FilterSolveDetailsByFreezeFromRepo(ctx, uc.deps.CompRepo, solves)
}

func (uc *UserUseCase) GetUserFails(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	if _, err := uc.deps.UserRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserFails - UserRepo.GetByID: %w", err)
	}

	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetFailsByUser(ctx, userID, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountFailsByUser(ctx, userID)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserFails: %w", err)
	}

	return result, nil
}

func (uc *UserUseCase) GetUserAwards(ctx context.Context, userID uuid.UUID) ([]*domain.Award, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserAwards - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return []*domain.Award{}, nil
	}

	if uc.deps.AwardRepo == nil {
		return []*domain.Award{}, nil
	}

	awards, err := uc.deps.AwardRepo.GetByTeamID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserAwards - AwardRepo.GetByTeamID: %w", err)
	}

	return awards, nil
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
		_ = uc.deps.EmailSender.SendVerificationEmail(ctx, user) //nolint:errcheck // best-effort
	}

	if newPassword != nil {
		err := uc.deps.JWTService.RevokeAllForUser(context.WithoutCancel(ctx), userID)
		if err != nil {
			uc.deps.Logger.WithError(err).Error("UserUseCase - UpdateProfile - RevokeAllForUser")
		}
	}

	return user, nil
}

func (uc *UserUseCase) GetMySubmissions(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetByUser(ctx, userID, nil, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountByUser(ctx, userID, nil)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetMySubmissions: %w", err)
	}

	return result, nil
}

func (d UserDeps) bcryptCost() int {
	if d.BcryptCost > 0 {
		return d.BcryptCost
	}

	return bcrypt.DefaultCost
}

func (uc *UserUseCase) bcryptCost() int {
	return uc.deps.bcryptCost()
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
