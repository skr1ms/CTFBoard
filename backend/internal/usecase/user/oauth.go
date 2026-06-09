package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// OAuthProviderToken is the normalized token returned by an OAuth provider adapter.
type OAuthProviderToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// OAuthProviderGateway owns provider-specific protocol details outside the
// usecase layer: authorization URL creation, code exchange, and profile fetch.
type OAuthProviderGateway interface {
	IsConfigured(provider string) bool
	AuthCodeURL(ctx context.Context, provider, state string) (string, error)
	Exchange(ctx context.Context, provider, code string) (*OAuthProviderToken, error)
	FetchUserProfile(ctx context.Context, provider, accessToken string) (*domain.OAuthUserProfile, error)
}

// OAuthProviderConfig holds credentials and redirect URL for a single OAuth provider.
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func (c OAuthProviderConfig) IsConfigured() bool {
	return c.ClientID != "" && c.ClientSecret != "" && c.RedirectURL != ""
}

// OAuthConfig holds the shared state secret and per-provider credentials.
type OAuthConfig struct {
	StateSecret string
	GitHub      OAuthProviderConfig
	Google      OAuthProviderConfig
}

const (
	oauthNonceBytes     = 16
	oauthExchangePrefix = "oauth_exchange:"
	oauthExchangeTTL    = 30 * time.Second
	oauthExchangeBytes  = 32
	oauthStatePartCount = 2
	usernameMaxLen      = 50
)

// ErrOAuthExchangeCodeNotFound is returned by the OAuth exchange store when a
// one-time exchange code is missing or already consumed.
var ErrOAuthExchangeCodeNotFound = errors.New("oauth exchange code not found")

// OAuthExchangeStore stores short-lived one-time OAuth exchange codes. It is a
// usecase-owned port so handlers never depend on Redis directly.
type OAuthExchangeStore interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	GetDel(ctx context.Context, key string) ([]byte, error)
}

type OAuthUseCase struct {
	deps        OAuthDeps
	stateSecret []byte
}

type OAuthDeps struct {
	UserRepo        repo.UserRepository
	OAuthRepo       repo.OAuthAccountRepository
	TM              repo.TransactionManager
	SettingsRepo    repo.SettingsRepository
	JWTService      jwtkit.Service
	ProviderGateway OAuthProviderGateway
	Cfg             OAuthConfig
	CompRepo        repo.CompetitionRepository
	SoloTeamCreator SoloTeamCreator
	ExchangeStore   OAuthExchangeStore
	CompParamUC     usecase.CompetitionParamUseCase
	Logger          logkit.Logger
}

var _ usecase.OAuthUseCase = (*OAuthUseCase)(nil)

func NewOAuthUseCase(deps OAuthDeps) *OAuthUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &OAuthUseCase{
		deps:        deps,
		stateSecret: []byte(deps.Cfg.StateSecret),
	}
}

// HandleCallback processes an OAuth provider callback. It exchanges the
// authorization code for an access token, fetches the user profile from the
// provider, and then follows one of three paths: (1) if an OAuth account record
// already exists for the provider/user-id pair, the existing user is logged in
// and the stored tokens are refreshed; (2) if no OAuth record exists but a local
// account with a matching verified email is found, the OAuth provider is linked
// to that account (a solo team is created if needed); (3) otherwise a new user
// account is registered. In all cases a JWT token pair is returned on success.
func (uc *OAuthUseCase) HandleCallback(ctx context.Context, provider, code string) (*usecase.TokenPair, error) {
	if err := uc.ensureProviderEnabled(ctx, provider); err != nil {
		return nil, fmt.Errorf("OAuthUseCase - HandleCallback - ensureProviderEnabled: %w", err)
	}

	token, err := uc.deps.ProviderGateway.Exchange(ctx, provider, code)
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - HandleCallback - Exchange (%s): %w", provider, err)
	}

	if token == nil || token.AccessToken == "" {
		return nil, fmt.Errorf("OAuthUseCase - HandleCallback - Exchange (%s): empty access token", provider)
	}

	profile, err := uc.deps.ProviderGateway.FetchUserProfile(ctx, provider, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - HandleCallback - FetchProfile (%s): %w", provider, err)
	}

	profile.Email = normalizeEmail(profile.Email)
	if profile.Email == "" {
		return nil, apperr.NewValidationErrorf("OAuth provider did not return an email address; make sure your email is public in your provider settings")
	}

	existing, err := uc.deps.OAuthRepo.GetByProvider(ctx, provider, profile.ID)
	if err != nil && !errors.Is(err, apperr.ErrOAuthAccountNotFound) {
		return nil, fmt.Errorf("OAuthUseCase - HandleCallback - GetByProvider: %w", err)
	}

	if existing != nil {
		return uc.loginExistingOAuthUser(ctx, existing, token, provider)
	}

	existingUser, err := uc.deps.UserRepo.GetByEmail(ctx, profile.Email)
	if err != nil && !errors.Is(err, apperr.ErrUserNotFound) {
		return nil, fmt.Errorf("OAuthUseCase - HandleCallback - GetByEmail: %w", err)
	}
	// Only auto-link when the local account's email is verified (or already OAuth-only)
	// An unverified local account could have been created with someone else's email,
	// which would let the attacker gain access to the victim's OAuth session
	if existingUser != nil && (existingUser.IsVerified || existingUser.PasswordHash == domain.OAuthOnlyPasswordSentinel) {
		return uc.linkOAuthToExistingUser(ctx, existingUser, profile, token, provider)
	}

	if existingUser != nil {
		// Local account exists but email is not yet verified. Do not auto-link, and
		// return a precise error instead of falling through to a registration conflict.
		return nil, apperr.ErrEmailNotVerified
	}

	return uc.registerNewOAuthUser(ctx, profile, token, provider)
}
