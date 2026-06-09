package user

import (
	"context"
	"runtime"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

// FieldValidator validates custom field values (e.g. on registration)
// Implemented by *settings.FieldValidator.
type FieldValidator interface {
	ValidateValues(ctx context.Context, entityType domain.EntityType, values map[uuid.UUID]any) (map[uuid.UUID]string, error)
	ValidateEditableValues(ctx context.Context, entityType domain.EntityType, values map[uuid.UUID]any) (map[uuid.UUID]string, error)
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
	CreatePersonal(ctx context.Context, params usecase.NotificationCreatePersonalParams) (*domain.UserNotification, error)
}

// APITokenRevoker revokes long-lived API tokens after credential resets.
type APITokenRevoker interface {
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type UserUseCase struct {
	deps      UserDeps
	bcryptSem chan struct{}      // limits concurrent bcrypt to avoid CPU saturation
	dummyHash []byte             // pre-computed hash used for constant-time login when user is not found
	profileSF singleflight.Group // deduplicates concurrent GetProfile calls for the same userID
}

const (
	bcryptWorkersPerCPU = 2
	bcryptMinWorkers    = 2
)

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
	FieldRepo                  repo.FieldRepository
	FieldValueRepo             repo.FieldValueRepository
	SettingsRepo               repo.SettingsRepository
	EmailSender                EmailVerificationSender
	APITokenRevoker            APITokenRevoker
	FailedLogin                FailedLoginTracker
	CompRepo                   repo.CompetitionRepository
	SoloTeamCreator            SoloTeamCreator
	Logger                     logkit.Logger
	UserCache                  cacheutil.UserCacheInvalidator
	ScoreboardCache            cacheutil.ScoreboardCacheInvalidator
	ChallengeListCache         cacheutil.ChallengeListCacheInvalidator
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
	n := max(runtime.NumCPU()*bcryptWorkersPerCPU, bcryptMinWorkers)

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

func (d UserDeps) bcryptCost() int {
	if d.BcryptCost > 0 {
		return d.BcryptCost
	}

	return bcrypt.DefaultCost
}

func (uc *UserUseCase) bcryptCost() int {
	return uc.deps.bcryptCost()
}
