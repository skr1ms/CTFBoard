package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-jwtkit"
	jwtMock "github.com/wahrwelt-kit/go-jwtkit/mock"
	"golang.org/x/oauth2"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/webapi"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type oauthTestDeps struct {
	UserRepo     *userMock.MockUserRepository
	OAuthRepo    *userMock.MockOAuthAccountRepository
	TM           *userMock.MockTransactionManager
	JWTService   *jwtMock.MockService
	SettingsRepo *userMock.MockSettingsRepository
	ProviderAPI  *userMock.MockOAuthProviderAPI
}

func newOAuthTestDeps(t *testing.T) *oauthTestDeps {
	t.Helper()
	return &oauthTestDeps{
		UserRepo:     userMock.NewMockUserRepository(t),
		OAuthRepo:    userMock.NewMockOAuthAccountRepository(t),
		TM:           userMock.NewMockTransactionManager(t),
		JWTService:   jwtMock.NewMockService(t),
		SettingsRepo: userMock.NewMockSettingsRepository(t),
		ProviderAPI:  userMock.NewMockOAuthProviderAPI(t),
	}
}

func (d *oauthTestDeps) createUseCase() *OAuthUseCase {
	return NewOAuthUseCase(OAuthDeps{
		UserRepo: d.UserRepo, OAuthRepo: d.OAuthRepo, TM: d.TM,
		SettingsRepo: d.SettingsRepo, JWTService: d.JWTService,
		Providers: map[string]webapi.OAuthProviderAPI{"github": d.ProviderAPI},
		Cfg: config.OAuth{
			StateSecret: "test-secret-for-oauth-state-1234",
			GitHub: config.OAuthProvider{
				ClientID: "test-client-id", ClientSecret: "test-client-secret",
				RedirectURL: "http://localhost:3000/auth/github/callback",
			},
		},
	})
}

func defaultOAuthSettings() *domain.Settings {
	return &domain.Settings{OAuthGithubEnabled: true, OAuthGoogleEnabled: false}
}

func newTestOAuthAccount(userID uuid.UUID, provider, providerUserID, accessToken string) *domain.OAuthAccount {
	return &domain.OAuthAccount{
		ID: uuid.New(), UserID: userID, Provider: provider,
		ProviderUserID: providerUserID, AccessToken: accessToken,
	}
}

func TestOAuthUseCase_GetAuthURL_Success(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(defaultOAuthSettings(), nil)

	authURL, state, err := d.createUseCase().GetAuthURL(context.Background(), "github")
	require.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.NotEmpty(t, state)
	assert.Contains(t, authURL, "github.com")
}

func TestOAuthUseCase_GetAuthURL_UnsupportedProvider(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(defaultOAuthSettings(), nil)

	_, _, err := d.createUseCase().GetAuthURL(context.Background(), "facebook")
	assert.ErrorIs(t, err, httperr.ErrOAuthUnsupportedProvider)
}

func TestOAuthUseCase_GetAuthURL_ProviderDisabled(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(defaultOAuthSettings(), nil)

	_, _, err := d.createUseCase().GetAuthURL(context.Background(), "google")
	assert.ErrorIs(t, err, httperr.ErrOAuthProviderDisabled)
}

func TestOAuthUseCase_ValidateState_Match(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{OAuthGithubEnabled: true}, nil)

	uc := d.createUseCase()
	_, state, err := uc.GetAuthURL(context.Background(), "github")
	require.NoError(t, err)

	assert.True(t, uc.ValidateState(state, state))
}

func TestOAuthUseCase_ValidateState_Mismatch(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)

	assert.False(t, d.createUseCase().ValidateState("cookie-state", "different-state"))
}

func TestOAuthUseCase_LoginExistingUser_Success(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	existingAcc := newTestOAuthAccount(userID, "github", "gh-123", "old-token")
	existingUser := &domain.User{ID: userID, Email: "user@gh.com", Username: "ghuser", Role: domain.RoleUser}
	tokenPair := &jwtkit.TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh", AccessExpiresAt: time.Now().Unix()}

	d.OAuthRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.UserRepo.EXPECT().GetByID(mock.Anything, userID).Return(existingUser, nil)
	d.JWTService.EXPECT().GenerateTokenPair(mock.Anything, userID, string(existingUser.Role)).Return(tokenPair, nil)

	pair, err := uc.loginExistingOAuthUser(context.Background(), existingAcc, &oauth2.Token{AccessToken: "new-token"}, "github")
	require.NoError(t, err)
	assert.Equal(t, "new-access", pair.AccessToken)
}

func TestOAuthUseCase_LoginExistingUser_UserRepoError(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	existingAcc := newTestOAuthAccount(uuid.New(), "github", "gh-456", "token")

	d.OAuthRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.UserRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("user not found"))

	_, err := uc.loginExistingOAuthUser(context.Background(), existingAcc, &oauth2.Token{AccessToken: "token"}, "github")
	assert.Error(t, err)
}

func TestOAuthUseCase_LoginExistingUser_WasInBannedTeam_Rejected(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	existingAcc := newTestOAuthAccount(userID, "github", "gh-123", "old-token")
	existingUser := &domain.User{ID: userID, Email: "user@gh.com", Username: "ghuser", Role: domain.RoleUser, WasInBannedTeam: true}

	d.OAuthRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.UserRepo.EXPECT().GetByID(mock.Anything, userID).Return(existingUser, nil)

	_, err := uc.loginExistingOAuthUser(context.Background(), existingAcc, &oauth2.Token{AccessToken: "new-token"}, "github")
	assert.ErrorIs(t, err, httperr.ErrInvalidCredentials)
}

func TestOAuthUseCase_RegisterNewUser_Success(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	profile := &webapi.OAuthUserProfile{ID: "gh-789", Email: "newuser@gh.com", Username: "newghuser"}
	tokenPair := &jwtkit.TokenPair{AccessToken: "access", RefreshToken: "refresh", AccessExpiresAt: time.Now().Unix()}

	d.TM.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: true}, nil).Once()
	d.UserRepo.EXPECT().GetByEmail(mock.Anything, "newuser@gh.com").Return(nil, httperr.ErrUserNotFound)
	d.UserRepo.EXPECT().GetByUsername(mock.Anything, "newghuser").Return(nil, httperr.ErrUserNotFound)
	d.UserRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, u *domain.User) {
		u.ID = uuid.New()
	})
	d.OAuthRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.JWTService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, string(domain.RoleUser)).Return(tokenPair, nil)

	pair, err := uc.registerNewOAuthUser(context.Background(), profile, &oauth2.Token{AccessToken: "gh-access"}, "github")
	require.NoError(t, err)
	assert.Equal(t, "access", pair.AccessToken)
}

func TestOAuthUseCase_RegisterNewUser_TxError(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	profile := &webapi.OAuthUserProfile{ID: "gh-err", Email: "err@gh.com", Username: "erruser"}

	d.TM.EXPECT().Run(mock.Anything, mock.Anything).Return(errors.New("tx error"))

	_, err := uc.registerNewOAuthUser(context.Background(), profile, &oauth2.Token{AccessToken: "token"}, "github")
	assert.Error(t, err)
}

func TestOAuthUseCase_RegisterNewUser_RegistrationClosed(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	profile := &webapi.OAuthUserProfile{ID: "gh-closed", Email: "new@gh.com", Username: "newuser"}

	d.TM.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: false}, nil).Once()

	_, err := uc.registerNewOAuthUser(context.Background(), profile, &oauth2.Token{AccessToken: "token"}, "github")
	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrRegistrationClosed)
}
