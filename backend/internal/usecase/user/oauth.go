package user

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

// OAuthProviderAPI abstracts fetching a user profile from an OAuth provider.
// Defined here (consumer side) per Clean Architecture: usecase layer does not
// depend on repo/webapi.
type OAuthProviderAPI interface {
	FetchUserProfile(ctx context.Context, accessToken string) (*domain.OAuthUserProfile, error)
}

// OAuthProviderConfig holds credentials and redirect URL for a single OAuth provider.
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// OAuthConfig holds the shared state secret and per-provider credentials.
type OAuthConfig struct {
	StateSecret string
	GitHub      OAuthProviderConfig
	Google      OAuthProviderConfig
}

const (
	oauthNonceBytes = 16
	usernameMaxLen  = 50
)

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
	Providers       map[string]OAuthProviderAPI
	Cfg             OAuthConfig
	CompRepo        repo.CompetitionRepository
	SoloTeamCreator SoloTeamCreator
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

// GetAuthURL builds the OAuth provider redirect URL with an HMAC-signed state parameter.
// The state is formatted as "<nonce_hex>.<hmac_sha256_hex>" where the nonce is
// cryptographically random; ValidateState verifies the signature on callback.
func (uc *OAuthUseCase) GetAuthURL(ctx context.Context, provider string) (authURL, state string, err error) {
	oauthCfg, err := uc.oauthConfig(ctx, provider)
	if err != nil {
		return "", "", fmt.Errorf("OAuthUseCase - GetAuthURL - oauthConfig: %w", err)
	}

	nonceHex, err := crypto.SecureRandomHex(oauthNonceBytes)
	if err != nil {
		return "", "", fmt.Errorf("OAuthUseCase - GetAuthURL: %w", err)
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return "", "", fmt.Errorf("OAuthUseCase - GetAuthURL - hex.DecodeString: %w", err)
	}

	mac := hmac.New(sha256.New, uc.stateSecret)
	mac.Write(nonce) //nolint:revive // hash.Write never returns error
	sig := hex.EncodeToString(mac.Sum(nil))
	state = nonceHex + "." + sig

	return oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline), state, nil
}

// ValidateState performs a two-step verification: first a constant-time comparison
// of the cookie and query state strings to detect tampering, then it re-derives the
// HMAC from the embedded nonce and checks it against the signature in the state,
// preventing CSRF attacks via forged OAuth callbacks.
func (uc *OAuthUseCase) ValidateState(cookieState, queryState string) bool {
	if !hmac.Equal([]byte(cookieState), []byte(queryState)) {
		return false
	}

	parts := strings.SplitN(queryState, ".", 2)
	if len(parts) != 2 {
		return false
	}

	nonce, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, uc.stateSecret)
	mac.Write(nonce) //nolint:revive // hash.Write never returns error
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(parts[1]), []byte(expectedSig))
}

// HandleCallback processes an OAuth provider callback. It exchanges the
// authorization code for an access token, fetches the user profile from the
// provider, and then follows one of three paths: (1) if an OAuth account record
// already exists for the provider/user-id pair, the existing user is logged in
// and the stored tokens are refreshed; (2) if no OAuth record exists but a local
// account with a matching verified email is found, the OAuth provider is linked
// to that account (a solo team is created if needed); (3) otherwise a new user
// account is registered. In all cases a JWT token pair is returned on success.
func (uc *OAuthUseCase) HandleCallback(ctx context.Context, provider, code string) (*jwtkit.TokenPair, error) {
	oauthCfg, err := uc.oauthConfig(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - HandleCallback - oauthConfig: %w", err)
	}

	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - HandleCallback - Exchange (%s): %w", provider, err)
	}

	providerAPI, ok := uc.deps.Providers[provider]
	if !ok {
		return nil, apperr.ErrOAuthUnsupportedProvider
	}

	profile, err := providerAPI.FetchUserProfile(ctx, token.AccessToken)
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
		// Local account exists but email is not yet verified - treat as a new registration
		// to avoid account takeover. The user should verify their email first
		return uc.registerNewOAuthUser(ctx, profile, token, provider)
	}

	return uc.registerNewOAuthUser(ctx, profile, token, provider)
}

// loginExistingOAuthUser refreshes the stored OAuth tokens for an already-linked
// account, then loads the user, checks ban status (IsBanned and WasInBannedTeam),
// and issues a new JWT token pair.
func (uc *OAuthUseCase) loginExistingOAuthUser(
	ctx context.Context,
	oauthAcc *domain.OAuthAccount,
	token *oauth2.Token,
	_ string,
) (*jwtkit.TokenPair, error) {
	oauthAcc.AccessToken = token.AccessToken

	rt := token.RefreshToken
	if rt != "" {
		oauthAcc.RefreshToken = &rt
	}

	if !token.Expiry.IsZero() {
		oauthAcc.ExpiresAt = &token.Expiry
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, oauthAcc.UserID)
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - loginExistingOAuthUser - GetByID: %w", err)
	}

	if user.IsBanned {
		return nil, apperr.ErrInvalidCredentials
	}

	if user.WasInBannedTeam && user.Role != domain.RoleAdmin {
		return nil, apperr.ErrInvalidCredentials
	}

	if err := uc.deps.OAuthRepo.Upsert(ctx, oauthAcc); err != nil {
		return nil, fmt.Errorf("OAuthUseCase - loginExistingOAuthUser - Upsert: %w", err)
	}

	pair, err := uc.deps.JWTService.GenerateTokenPair(ctx, user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - loginExistingOAuthUser - GenerateTokenPair: %w", err)
	}

	return pair, nil
}

// linkOAuthToExistingUser attaches an OAuth provider to an existing account (same email). Returns JWT for that user.
func (uc *OAuthUseCase) linkOAuthToExistingUser(
	ctx context.Context,
	existingUser *domain.User,
	profile *domain.OAuthUserProfile,
	token *oauth2.Token,
	provider string,
) (*jwtkit.TokenPair, error) {
	if existingUser.IsBanned {
		return nil, apperr.ErrInvalidCredentials
	}

	if existingUser.WasInBannedTeam && existingUser.Role != domain.RoleAdmin {
		return nil, apperr.ErrInvalidCredentials
	}

	oauthAcc := &domain.OAuthAccount{
		UserID:         existingUser.ID,
		Provider:       provider,
		ProviderUserID: profile.ID,
		AccessToken:    token.AccessToken,
	}
	if token.RefreshToken != "" {
		oauthAcc.RefreshToken = &token.RefreshToken
	}

	if !token.Expiry.IsZero() {
		oauthAcc.ExpiresAt = &token.Expiry
	}

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		err := uc.deps.OAuthRepo.Upsert(ctx, oauthAcc)
		if err != nil {
			return fmt.Errorf("OAuthUseCase - linkOAuthToExistingUser - OAuthRepo.Upsert: %w", err)
		}
		// In solo_only mode an existing user without a team (e.g. registered before
		// the mode was set to solo_only) must receive a solo team on first OAuth login
		// so they can submit flags
		if existingUser.TeamID == nil {
			if err := ensureSoloTeamIfRequired(ctx, uc.deps.CompRepo, uc.deps.SoloTeamCreator, existingUser); err != nil {
				return fmt.Errorf("OAuthUseCase - linkOAuthToExistingUser - ensureSoloTeamIfRequired: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - linkOAuthToExistingUser - Transaction: %w", err)
	}

	pair, err := uc.deps.JWTService.GenerateTokenPair(ctx, existingUser.ID, string(existingUser.Role))
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - linkOAuthToExistingUser - GenerateTokenPair: %w", err)
	}

	return pair, nil
}

// registerNewOAuthUser creates a brand-new user account from the OAuth provider
// profile data inside a single transaction. It checks that registration is open,
// verifies that no local account already exists for the provider email, and
// resolves a unique username by first trying the provider-supplied name, then
// a provider-scoped fallback that embeds the provider user ID to avoid collisions
// The user is created with the email pre-verified and an OAuth-only password
// sentinel. In solo_only competition mode a solo team is created within the same
// transaction. On success a JWT token pair is issued.
func (uc *OAuthUseCase) registerNewOAuthUser(
	ctx context.Context,
	profile *domain.OAuthUserProfile,
	token *oauth2.Token,
	provider string,
) (*jwtkit.TokenPair, error) {
	user := &domain.User{
		Email:        profile.Email,
		PasswordHash: domain.OAuthOnlyPasswordSentinel,
		Role:         domain.RoleUser,
		IsVerified:   true,
	}

	oauthAcc := &domain.OAuthAccount{
		Provider:       provider,
		ProviderUserID: profile.ID,
		AccessToken:    token.AccessToken,
	}
	if token.RefreshToken != "" {
		oauthAcc.RefreshToken = &token.RefreshToken
	}

	if !token.Expiry.IsZero() {
		oauthAcc.ExpiresAt = &token.Expiry
	}

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		keyEmail := registrationAdvisoryKey("reg:email:", profile.Email)
		if err := uc.deps.UserRepo.AcquireAdvisoryLock(ctx, keyEmail); err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - AcquireAdvisoryLock(email): %w", err)
		}

		settings, err := uc.deps.SettingsRepo.Get(ctx)
		if err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - SettingsRepo.Get: %w", err)
		}

		if !settings.RegistrationOpen {
			return apperr.ErrRegistrationClosed
		}

		existing, err := uc.deps.UserRepo.GetByEmail(ctx, profile.Email)
		if err != nil && !errors.Is(err, apperr.ErrUserNotFound) {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - UserRepo.GetByEmail: %w", err)
		}

		if existing != nil {
			return apperr.ErrUserAlreadyExists
		}

		username, err := uc.resolveUsername(ctx, profile.Username, provider, profile.ID)
		if err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - resolveUsername: %w", err)
		}

		keyUsername := registrationAdvisoryKey("reg:username:", username)
		if keyUsername != keyEmail {
			if err := uc.deps.UserRepo.AcquireAdvisoryLock(ctx, keyUsername); err != nil {
				return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - AcquireAdvisoryLock(username): %w", err)
			}
		}

		user.Username = username
		if err := uc.deps.UserRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - UserRepo.Create: %w", err)
		}

		oauthAcc.UserID = user.ID
		if err := uc.deps.OAuthRepo.Upsert(ctx, oauthAcc); err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - OAuthRepo.Upsert: %w", err)
		}
		// Solo team creation is inside the transaction so a failure rolls back
		// the entire registration. CreateSoloTeamForNewUser reuses this tx
		if err := ensureSoloTeamIfRequired(ctx, uc.deps.CompRepo, uc.deps.SoloTeamCreator, user); err != nil {
			return fmt.Errorf("OAuthUseCase - registerNewOAuthUser - ensureSoloTeamIfRequired: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - registerNewOAuthUser - Transaction: %w", err)
	}

	pair, err := uc.deps.JWTService.GenerateTokenPair(ctx, user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - registerNewOAuthUser - GenerateTokenPair: %w", err)
	}

	return pair, nil
}

func truncateUsername(s string) string {
	runes := []rune(s)
	if len(runes) <= usernameMaxLen {
		return s
	}

	return string(runes[:usernameMaxLen])
}

// resolveUsername picks a username (within the current ctx transaction) to avoid TOCTOU
// between the uniqueness check and the subsequent Create call.
func (uc *OAuthUseCase) resolveUsername(ctx context.Context, desired, provider, providerID string) (string, error) {
	if desired == "" {
		desired = provider + "-user"
	}

	desired = truncateUsername(desired)

	_, err := uc.deps.UserRepo.GetByUsername(ctx, desired)
	if errors.Is(err, apperr.ErrUserNotFound) {
		return desired, nil
	}

	if err != nil {
		return "", fmt.Errorf("OAuthUseCase - resolveUsername - UserRepo.GetByUsername: %w", err)
	}

	// desired is taken - use provider-scoped fallback that includes the unique provider ID
	fallback := truncateUsername(fmt.Sprintf("%s-%s-%s", desired, provider, providerID))

	_, err = uc.deps.UserRepo.GetByUsername(ctx, fallback)
	if errors.Is(err, apperr.ErrUserNotFound) {
		return fallback, nil
	}

	if err != nil {
		return "", fmt.Errorf("OAuthUseCase - resolveUsername - UserRepo.GetByUsername fallback: %w", err)
	}

	return "", apperr.ErrUsernameTaken
}

// oauthConfig reads the current app settings to check whether the provider is
// enabled, then constructs the provider-specific *oauth2.Config with the
// credentials and scopes from the application config. Returns
// ErrOAuthProviderDisabled when the provider is toggled off in settings.
func (uc *OAuthUseCase) oauthConfig(ctx context.Context, provider string) (*oauth2.Config, error) {
	settings, err := uc.deps.SettingsRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("OAuthUseCase - oauthConfig - GetSettings: %w", err)
	}

	switch provider {
	case "github":
		if !settings.OAuthGithubEnabled {
			return nil, apperr.ErrOAuthProviderDisabled
		}

		return &oauth2.Config{
			ClientID:     uc.deps.Cfg.GitHub.ClientID,
			ClientSecret: uc.deps.Cfg.GitHub.ClientSecret,
			RedirectURL:  uc.deps.Cfg.GitHub.RedirectURL,
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     github.Endpoint,
		}, nil

	case "google":
		if !settings.OAuthGoogleEnabled {
			return nil, apperr.ErrOAuthProviderDisabled
		}

		return &oauth2.Config{
			ClientID:     uc.deps.Cfg.Google.ClientID,
			ClientSecret: uc.deps.Cfg.Google.ClientSecret,
			RedirectURL:  uc.deps.Cfg.Google.RedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}, nil

	default:
		return nil, apperr.ErrOAuthUnsupportedProvider
	}
}
