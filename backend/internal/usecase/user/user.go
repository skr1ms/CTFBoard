package user

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// FieldValidator validates custom field values (e.g. on registration).
// Implemented by *settings.FieldValidator.
type FieldValidator interface {
	ValidateValues(ctx context.Context, entityType entity.EntityType, values map[uuid.UUID]string) error
}

// EmailVerificationSender sends verification email (e.g. usecase.EmailUseCase).
// Optional; when set, used to send verification after email change in UpdateProfile.
type EmailVerificationSender interface {
	SendVerificationEmail(ctx context.Context, user *entity.User) error
}

// FailedLoginTracker counts failed logins per email for lockout. Optional.
type FailedLoginTracker interface {
	IsLocked(ctx context.Context, email string) (bool, error)
	RecordFailed(ctx context.Context, email string) error
}

// SoloTeamCreator creates a solo team for a user. Used to auto-create solo teams on registration when competition mode is solo_only.
type SoloTeamCreator interface {
	CreateSoloTeamForNewUser(ctx context.Context, userID uuid.UUID) (*entity.Team, error)
}

// UserCacheInvalidator evicts a cached user entry so ban/unban takes effect immediately.
// Implemented by *cache.UserCacheService.
type UserCacheInvalidator interface {
	InvalidateUser(ctx context.Context, userID uuid.UUID)
}

type UserUseCase struct {
	deps      UserDeps
	bcryptSem chan struct{} // limits concurrent bcrypt to avoid CPU saturation
}

type UserDeps struct {
	UserRepo        repo.UserRepository
	TeamRepo        repo.TeamRepository
	SolveRepo       repo.SolveRepository
	SubmissionRepo  repo.SubmissionRepository
	AwardRepo       repo.AwardRepository
	TM              repo.TransactionManager
	JWTService      jwt.Service
	FieldValidator  FieldValidator
	FieldValueRepo  repo.FieldValueRepository
	SettingsRepo    repo.SettingsRepository
	EmailSender     EmailVerificationSender
	FailedLogin     FailedLoginTracker
	CompRepo        repo.CompetitionRepository
	SoloTeamCreator SoloTeamCreator
	Logger          logger.Logger
	UserCache       UserCacheInvalidator
	ScoreboardCache cache.ScoreboardCacheInvalidator
}

var _ usecase.UserUseCase = (*UserUseCase)(nil)

func NewUserUseCase(deps UserDeps) *UserUseCase {
	n := runtime.NumCPU() * 2
	if n < 2 {
		n = 2
	}
	if deps.Logger == nil {
		deps.Logger = logger.Noop()
	}
	return &UserUseCase{deps: deps, bcryptSem: make(chan struct{}, n)}
}

//nolint:gocognit,gocyclo // registration flow with validation branches
func (uc *UserUseCase) Register(ctx context.Context, username, email, password string, customFields map[string]string) (*entity.User, error) {
	email = normalizeEmail(email)
	if err := uc.registerValidateCustomFields(ctx, customFields); err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - registerValidateCustomFields: %w", err)
	}
	if uc.deps.SettingsRepo != nil {
		settings, err := uc.deps.SettingsRepo.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("UserUseCase - Register - SettingsRepo.Get: %w", err)
		}
		if !settings.RegistrationOpen {
			return nil, httperr.ErrRegistrationClosed
		}
	}
	uc.bcryptSem <- struct{}{}
	defer func() { <-uc.bcryptSem }()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - GenerateFromPassword: %w", err)
	}
	user := &entity.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         entity.RoleUser,
	}
	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.registerCheckUniqueness(ctx, username, email); err != nil {
			return fmt.Errorf("UserUseCase - Register - registerCheckUniqueness: %w", err)
		}
		if err := uc.deps.UserRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("UserUseCase - Register - UserRepo.Create: %w", err)
		}
		if uc.deps.FieldValueRepo != nil && len(customFields) > 0 {
			if err := uc.deps.FieldValueRepo.SetValues(ctx, user.ID, customFields); err != nil {
				return fmt.Errorf("UserUseCase - Register - FieldValueRepo.SetValues: %w", err)
			}
		}
		// Solo team creation is inside the transaction so that a failure rolls
		// back the entire registration rather than leaving a user without a team.
		// CreateSoloTeamForNewUser internally calls TM.Run which reuses this tx.
		if uc.deps.CompRepo != nil && uc.deps.SoloTeamCreator != nil {
			comp, err := uc.deps.CompRepo.Get(ctx)
			if err != nil {
				return fmt.Errorf("UserUseCase - Register - CompRepo.Get: %w", err)
			}
			if comp.Mode == entity.ModeSoloOnly {
				team, err := uc.deps.SoloTeamCreator.CreateSoloTeamForNewUser(ctx, user.ID)
				if err != nil {
					return fmt.Errorf("UserUseCase - Register - SoloTeamCreator.CreateSoloTeamForNewUser: %w", err)
				}
				user.TeamID = &team.ID
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - Register - TM.Run: %w", err)
	}
	return user, nil
}

func (uc *UserUseCase) registerValidateCustomFields(ctx context.Context, customFields map[string]string) error {
	if len(customFields) == 0 || uc.deps.FieldValidator == nil {
		return nil
	}
	fieldValues := make(map[uuid.UUID]string)
	for k, v := range customFields {
		id, err := uuid.Parse(k)
		if err != nil {
			return fmt.Errorf("UserUseCase - registerValidateCustomFields - uuid.Parse: %w", err)
		}
		fieldValues[id] = v
	}
	if err := uc.deps.FieldValidator.ValidateValues(ctx, entity.EntityTypeUser, fieldValues); err != nil {
		return fmt.Errorf("UserUseCase - registerValidateCustomFields - FieldValidator.ValidateValues: %w", err)
	}
	return nil
}

func (uc *UserUseCase) registerCheckUniqueness(ctx context.Context, username, email string) error {
	existing, err := uc.deps.UserRepo.GetByUsername(ctx, username)
	if err == nil && existing != nil {
		return fmt.Errorf("UserUseCase - registerCheckUniqueness - username taken: %w", httperr.ErrUserAlreadyExists)
	}
	if err != nil && !errors.Is(err, httperr.ErrUserNotFound) {
		return fmt.Errorf("UserUseCase - registerCheckUniqueness - UserRepo.GetByUsername: %w", err)
	}
	existing, err = uc.deps.UserRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return fmt.Errorf("UserUseCase - registerCheckUniqueness - email taken: %w", httperr.ErrUserAlreadyExists)
	}
	if err != nil && !errors.Is(err, httperr.ErrUserNotFound) {
		return fmt.Errorf("UserUseCase - registerCheckUniqueness - UserRepo.GetByEmail: %w", err)
	}
	return nil
}

func (uc *UserUseCase) Login(ctx context.Context, email, password string) (*jwt.TokenPair, error) {
	email = normalizeEmail(email)
	if uc.deps.FailedLogin != nil {
		locked, err := uc.deps.FailedLogin.IsLocked(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("UserUseCase - Login - FailedLogin.IsLocked: %w", err)
		}
		if locked {
			return nil, httperr.ErrTooManyRequests
		}
	}

	user, err := uc.deps.UserRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, httperr.ErrUserNotFound) {
			uc.recordFailedLogin(ctx, email)
			return nil, httperr.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("UserUseCase - Login - UserRepo.GetByEmail: %w", err)
	}

	if user.IsBanned {
		uc.recordFailedLogin(ctx, email)
		return nil, httperr.ErrInvalidCredentials
	}

	if user.PasswordHash == "" || user.PasswordHash == entity.OAuthOnlyPasswordSentinel {
		uc.recordFailedLogin(ctx, email)
		return nil, httperr.ErrInvalidCredentials
	}
	uc.bcryptSem <- struct{}{}
	defer func() { <-uc.bcryptSem }()
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		uc.recordFailedLogin(ctx, email)
		return nil, httperr.ErrInvalidCredentials
	}

	tokenPair, err := uc.deps.JWTService.GenerateTokenPair(user.ID, user.Email, user.Username, user.Role)
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

func (uc *UserUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.User, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetByID - UserRepo.GetByID: %w", err)
	}
	return user, nil
}

func (uc *UserUseCase) GetProfile(ctx context.Context, userID uuid.UUID) (*usecase.UserProfile, error) {
	var user *entity.User
	var solves []*entity.Solve

	g, gCtx := errgroup.WithContext(ctx)
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
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("UserUseCase - GetProfile - errgroup.Wait: %w", err)
	}

	user.PasswordHash = ""

	return &usecase.UserProfile{
		User:   user,
		Solves: solves,
	}, nil
}

func (uc *UserUseCase) ListUsers(ctx context.Context, search *string, field string, page, perPage int) (*usecase.Paginated[*entity.User], error) {
	offset := (page - 1) * perPage
	var users []*entity.User
	var total int64

	g, gCtx := errgroup.WithContext(ctx)
	if field == "ip" && search != nil && *search != "" {
		g.Go(func() error {
			var err error
			users, err = uc.deps.UserRepo.SearchByIP(gCtx, *search, perPage, offset)
			if err != nil {
				return fmt.Errorf("UserUseCase - ListUsers - UserRepo.SearchByIP: %w", err)
			}
			return nil
		})
		g.Go(func() error {
			var err error
			total, err = uc.deps.UserRepo.CountSearchByIP(gCtx, *search)
			if err != nil {
				return fmt.Errorf("UserUseCase - ListUsers - UserRepo.CountSearchByIP: %w", err)
			}
			return nil
		})
	} else {
		g.Go(func() error {
			var err error
			users, err = uc.deps.UserRepo.Search(gCtx, search, perPage, offset)
			if err != nil {
				return fmt.Errorf("UserUseCase - ListUsers - UserRepo.Search: %w", err)
			}
			return nil
		})
		g.Go(func() error {
			var err error
			total, err = uc.deps.UserRepo.CountSearch(gCtx, search)
			if err != nil {
				return fmt.Errorf("UserUseCase - ListUsers - UserRepo.CountSearch: %w", err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("UserUseCase - ListUsers - errgroup.Wait: %w", err)
	}

	return usecase.NewPaginated(users, total, page, perPage), nil
}

func (uc *UserUseCase) GetUserSolves(ctx context.Context, userID uuid.UUID) ([]*entity.SolveWithDetails, error) {
	if _, err := uc.deps.UserRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserSolves - UserRepo.GetByID: %w", err)
	}
	solves, err := uc.deps.SolveRepo.GetByUserIDWithDetails(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserSolves - SolveRepo.GetByUserIDWithDetails: %w", err)
	}
	return solves, nil
}

func (uc *UserUseCase) GetUserFails(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.Paginated[*entity.SubmissionWithDetails], error) {
	if _, err := uc.deps.UserRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserFails - UserRepo.GetByID: %w", err)
	}
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
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

func (uc *UserUseCase) GetUserAwards(ctx context.Context, userID uuid.UUID) ([]*entity.Award, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserAwards - UserRepo.GetByID: %w", err)
	}
	if user.TeamID == nil {
		return []*entity.Award{}, nil
	}
	if uc.deps.AwardRepo == nil {
		return []*entity.Award{}, nil
	}
	awards, err := uc.deps.AwardRepo.GetByTeamID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserAwards - AwardRepo.GetByTeamID: %w", err)
	}
	return awards, nil
}

func (uc *UserUseCase) AdminCreate(ctx context.Context, username, email, password, role string) (*entity.User, error) {
	email = normalizeEmail(email)
	uc.bcryptSem <- struct{}{}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	<-uc.bcryptSem
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - AdminCreate - GenerateFromPassword: %w", err)
	}
	if role == "" {
		role = entity.RoleUser
	}
	if role != entity.RoleUser && role != entity.RoleAdmin {
		return nil, httperr.NewValidationErrorf("invalid role %q: must be one of [user, admin]", role)
	}
	user := &entity.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         role,
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
			if comp.Mode == entity.ModeSoloOnly {
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

func (uc *UserUseCase) AdminUpdate(ctx context.Context, userID uuid.UUID, username, email, role, password *string, isVerified *bool) (*entity.User, error) {
	if email != nil {
		norm := normalizeEmail(*email)
		email = &norm
	}
	if role != nil && *role != entity.RoleUser && *role != entity.RoleAdmin {
		return nil, httperr.NewValidationErrorf("invalid role %q: must be one of [user, admin]", *role)
	}
	// Bcrypt outside the transaction to avoid holding the TX open during CPU work.
	var passwordHash *string
	if password != nil {
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
		uc.deps.UserCache.InvalidateUser(ctx, userID)
	}
	return user, nil
}

func (uc *UserUseCase) AdminDelete(ctx context.Context, userID uuid.UUID) error {
	if err := uc.deps.UserRepo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("UserUseCase - AdminDelete - UserRepo.Delete: %w", err)
	}
	return nil
}

func (uc *UserUseCase) BanUser(ctx context.Context, userID uuid.UUID, reason string, actorID uuid.UUID) error {
	if userID == actorID {
		return httperr.ErrAccessDenied
	}
	var target *entity.User
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			return fmt.Errorf("UserUseCase - BanUser - UserRepo.Lock: %w", err)
		}
		u, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("UserUseCase - BanUser - UserRepo.GetByID: %w", err)
		}
		if u.Role == entity.RoleAdmin {
			return httperr.ErrAccessDenied
		}
		if err := uc.deps.UserRepo.Ban(ctx, userID, reason); err != nil {
			return fmt.Errorf("UserUseCase - BanUser - UserRepo.Ban: %w", err)
		}
		target = u
		return nil
	}); err != nil {
		return fmt.Errorf("UserUseCase - BanUser - TM.Run: %w", err)
	}
	if uc.deps.UserCache != nil {
		uc.deps.UserCache.InvalidateUser(ctx, userID)
	}
	uc.hideSoloTeamIfNeeded(ctx, target)
	return nil
}

func (uc *UserUseCase) UnbanUser(ctx context.Context, userID, actorID uuid.UUID) error {
	if userID == actorID {
		return httperr.ErrAccessDenied
	}
	var target *entity.User
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			return fmt.Errorf("UserUseCase - UnbanUser - UserRepo.Lock: %w", err)
		}
		u, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("UserUseCase - UnbanUser - UserRepo.GetByID: %w", err)
		}
		if err := uc.deps.UserRepo.Unban(ctx, userID); err != nil {
			return fmt.Errorf("UserUseCase - UnbanUser - UserRepo.Unban: %w", err)
		}
		target = u
		return nil
	}); err != nil {
		return fmt.Errorf("UserUseCase - UnbanUser - TM.Run: %w", err)
	}
	if uc.deps.UserCache != nil {
		uc.deps.UserCache.InvalidateUser(ctx, userID)
	}
	uc.showSoloTeamIfNeeded(ctx, target)
	return nil
}

// hideSoloTeamIfNeeded hides a solo team when its sole member is banned,
// so the team disappears from the scoreboard and team listings without permanently deleting data.
// This applies to both auto-created and manually created solo teams since a solo team always
// has exactly one member; if that member is banned the team should not appear on the scoreboard.
func (uc *UserUseCase) hideSoloTeamIfNeeded(ctx context.Context, user *entity.User) {
	if user.TeamID == nil || uc.deps.TeamRepo == nil {
		return
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil || !team.IsSolo || team.IsHidden {
		return
	}
	if err := uc.deps.TeamRepo.SetHidden(ctx, team.ID, true); err != nil {
		uc.deps.Logger.WithError(err).Error("UserUseCase - hideSoloTeamIfNeeded - SetHidden")
		return
	}
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, team.ID)
	}
}

// showSoloTeamIfNeeded restores visibility of a solo team when its member is unbanned.
// This applies to both auto-created and manually created solo teams.
func (uc *UserUseCase) showSoloTeamIfNeeded(ctx context.Context, user *entity.User) {
	if user.TeamID == nil || uc.deps.TeamRepo == nil {
		return
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil || !team.IsSolo || !team.IsHidden {
		return
	}
	if err := uc.deps.TeamRepo.SetHidden(ctx, team.ID, false); err != nil {
		uc.deps.Logger.WithError(err).Error("UserUseCase - showSoloTeamIfNeeded - SetHidden")
		return
	}
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, team.ID)
	}
}

// profileCheckUniqueness verifies that the requested username and email are not
// already taken by another user. currentUsername and currentEmail are the
// caller's existing values; fields that did not change are skipped to avoid
// spurious conflicts.
func (uc *UserUseCase) profileCheckUniqueness(ctx context.Context, currentUsername, currentEmail string, username, email *string) error {
	if username != nil && *username != currentUsername {
		existing, err := uc.deps.UserRepo.GetByUsername(ctx, *username)
		if err == nil && existing != nil {
			return httperr.ErrUserAlreadyExists
		}
		if err != nil && !errors.Is(err, httperr.ErrUserNotFound) {
			return fmt.Errorf("UserUseCase - profileCheckUniqueness - UserRepo.GetByUsername: %w", err)
		}
	}
	if email != nil && *email != currentEmail {
		existing, err := uc.deps.UserRepo.GetByEmail(ctx, *email)
		if err == nil && existing != nil {
			return httperr.ErrUserAlreadyExists
		}
		if err != nil && !errors.Is(err, httperr.ErrUserNotFound) {
			return fmt.Errorf("UserUseCase - profileCheckUniqueness - UserRepo.GetByEmail: %w", err)
		}
	}
	return nil
}

//nolint:gocognit,gocyclo // profile update branches
func (uc *UserUseCase) UpdateProfile(ctx context.Context, userID uuid.UUID, username, email, currentPassword, newPassword *string) (*entity.User, error) {
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
			return nil, httperr.ErrInvalidCredentials
		}
		if err := bcrypt.CompareHashAndPassword([]byte(current.PasswordHash), []byte(*currentPassword)); err != nil {
			return nil, httperr.ErrInvalidCredentials
		}
	}
	var passwordHash *string
	if newPassword != nil {
		uc.bcryptSem <- struct{}{}
		hash, err := bcrypt.GenerateFromPassword([]byte(*newPassword), bcrypt.DefaultCost)
		<-uc.bcryptSem
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
		// Re-fetch inside the transaction so uniqueness checks are consistent.
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
			if err := uc.deps.UserRepo.SetUnverified(ctx, userID); err != nil {
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
	return user, nil
}

func (uc *UserUseCase) GetMySubmissions(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.Paginated[*entity.SubmissionWithDetails], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetByUser(ctx, userID, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountByUser(ctx, userID)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetMySubmissions: %w", err)
	}
	return result, nil
}
