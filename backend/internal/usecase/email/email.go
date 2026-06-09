package email

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// JWTRevoker is the subset of jwtkit.Service used to invalidate all active
// tokens for a user after a password reset.
type JWTRevoker interface {
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

const (
	emailTokenBytes             = 32
	defaultAppName              = "CTF Platform"
	emailPlaceholderCTFName     = "{ctf_name}"
	emailPlaceholderURL         = "{url}"
	defaultVerificationSubject  = "Verify your email - " + emailPlaceholderCTFName
	defaultVerificationBody     = "Follow the link to verify: " + emailPlaceholderURL
	defaultPasswordResetSubject = "Password reset - " + emailPlaceholderCTFName
	defaultPasswordResetBody    = "Follow the link to reset password: " + emailPlaceholderURL
	defaultPasswordMinLength    = 8
	bcryptWorkersPerCPU         = 2
	bcryptMinWorkers            = 2
)

func substitute(s string, m map[string]string) string {
	for k, v := range m {
		s = strings.ReplaceAll(s, k, v)
	}

	return s
}

type EmailUseCase struct {
	deps      EmailDeps
	bcryptSem chan struct{}
}

type ConfigGetter interface {
	GetString(ctx context.Context, key, defaultVal string) string
	GetInt(ctx context.Context, key string, defaultVal int) int
}

type EmailDeps struct {
	UserRepo        UserRepository
	TokenRepo       VerificationTokenRepository
	TM              TransactionManager
	Mailer          Mailer
	ConfigUC        ConfigGetter
	JWTRevoker      JWTRevoker
	APITokenRevoker APITokenRevoker
	VerifyTTL       time.Duration
	ResetTTL        time.Duration
	FrontendURL     string
	Enabled         bool
	Logger          logkit.Logger
	BcryptCost      int // 0 = bcrypt.DefaultCost; set to bcrypt.MinCost in tests
}

var _ usecase.EmailUseCase = (*EmailUseCase)(nil)

func NewEmailUseCase(deps EmailDeps) *EmailUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	n := max(runtime.NumCPU()*bcryptWorkersPerCPU, bcryptMinWorkers)

	return &EmailUseCase{deps: deps, bcryptSem: make(chan struct{}, n)}
}

func (uc *EmailUseCase) IsEnabled() bool {
	return uc.deps.Enabled
}

func (d EmailDeps) bcryptCostValue() int {
	if d.BcryptCost > 0 {
		return d.BcryptCost
	}

	return bcrypt.DefaultCost
}

func (uc *EmailUseCase) hashPassword(ctx context.Context, password string) (string, error) {
	select {
	case uc.bcryptSem <- struct{}{}:
		defer func() { <-uc.bcryptSem }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), uc.deps.bcryptCostValue())
	if err != nil {
		return "", fmt.Errorf("EmailUseCase - hashPassword - GenerateFromPassword: %w", err)
	}

	return string(hash), nil
}
