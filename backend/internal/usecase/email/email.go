package email

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

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
)

func substitute(s string, m map[string]string) string {
	for k, v := range m {
		s = strings.ReplaceAll(s, k, v)
	}

	return s
}

type EmailUseCase struct {
	deps EmailDeps
}

type ConfigGetter interface {
	GetString(ctx context.Context, key, defaultVal string) string
	GetInt(ctx context.Context, key string, defaultVal int) int
}

type EmailDeps struct {
	UserRepo    UserRepository
	TokenRepo   VerificationTokenRepository
	TM          TransactionManager
	Mailer      Mailer
	ConfigUC    ConfigGetter
	JWTRevoker  JWTRevoker
	VerifyTTL   time.Duration
	ResetTTL    time.Duration
	FrontendURL string
	Enabled     bool
	Logger      logkit.Logger
	BcryptCost  int // 0 = bcrypt.DefaultCost; set to bcrypt.MinCost in tests
}

var _ usecase.EmailUseCase = (*EmailUseCase)(nil)

func NewEmailUseCase(deps EmailDeps) *EmailUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &EmailUseCase{deps: deps}
}

func (uc *EmailUseCase) IsEnabled() bool {
	return uc.deps.Enabled
}
