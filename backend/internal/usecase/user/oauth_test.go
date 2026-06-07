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

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
)

type memoryOAuthExchangeStore struct {
	values map[string][]byte
}

func newMemoryOAuthExchangeStore() *memoryOAuthExchangeStore {
	return &memoryOAuthExchangeStore{values: make(map[string][]byte)}
}

func (s *memoryOAuthExchangeStore) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	s.values[key] = value

	return nil
}

func (s *memoryOAuthExchangeStore) GetDel(_ context.Context, key string) ([]byte, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, ErrOAuthExchangeCodeNotFound
	}

	delete(s.values, key)

	return value, nil
}

type oauthTestDeps struct {
	UserRepo     *userMock.MockUserRepository
	OAuthRepo    *userMock.MockOAuthAccountRepository
	TM           *userMock.MockTransactionManager
	JWTService   *jwtMock.MockService
	SettingsRepo *userMock.MockSettingsRepository
	Gateway      *fakeOAuthProviderGateway
	CompParamUC  usecase.CompetitionParamUseCase
}

type fakeOAuthProviderGateway struct {
	configured map[string]bool
	authURL    string
	authErr    error
	token      *OAuthProviderToken
	tokenErr   error
	profile    *domain.OAuthUserProfile
	profileErr error
}

func newFakeOAuthProviderGateway() *fakeOAuthProviderGateway {
	return &fakeOAuthProviderGateway{
		configured: map[string]bool{"github": true},
		authURL:    "https://github.com/login/oauth/authorize?state=test",
		token:      &OAuthProviderToken{AccessToken: "provider-access-token"},
		profile:    &domain.OAuthUserProfile{ID: "gh-1", Email: "user@example.com", Username: "ghuser"},
	}
}

func (g *fakeOAuthProviderGateway) IsConfigured(provider string) bool {
	return g.configured[provider]
}

func (g *fakeOAuthProviderGateway) AuthCodeURL(context.Context, string, string) (string, error) {
	return g.authURL, g.authErr
}

func (g *fakeOAuthProviderGateway) Exchange(context.Context, string, string) (*OAuthProviderToken, error) {
	return g.token, g.tokenErr
}

func (g *fakeOAuthProviderGateway) FetchUserProfile(context.Context, string, string) (*domain.OAuthUserProfile, error) {
	return g.profile, g.profileErr
}

func newOAuthTestDeps(t *testing.T) *oauthTestDeps {
	t.Helper()

	return &oauthTestDeps{
		UserRepo:     userMock.NewMockUserRepository(t),
		OAuthRepo:    userMock.NewMockOAuthAccountRepository(t),
		TM:           userMock.NewMockTransactionManager(t),
		JWTService:   jwtMock.NewMockService(t),
		SettingsRepo: userMock.NewMockSettingsRepository(t),
		Gateway:      newFakeOAuthProviderGateway(),
	}
}

func TestOAuthUseCase_ExchangeCode_RoundTripAndReplayRejected(t *testing.T) {
	t.Parallel()

	store := newMemoryOAuthExchangeStore()
	uc := NewOAuthUseCase(OAuthDeps{
		ExchangeStore: store,
		Cfg:           OAuthConfig{StateSecret: "test-secret-for-oauth-state-1234"},
	})

	pair := &usecase.TokenPair{
		AccessToken:      "access",
		RefreshToken:     "refresh",
		AccessExpiresAt:  123,
		RefreshExpiresAt: 456,
	}

	code, err := uc.IssueExchangeCode(context.Background(), pair)
	require.NoError(t, err)
	require.Len(t, code, oauthExchangeBytes*2)

	got, err := uc.ConsumeExchangeCode(context.Background(), code)
	require.NoError(t, err)
	assert.Equal(t, pair.AccessToken, got.AccessToken)
	assert.Equal(t, pair.RefreshToken, got.RefreshToken)
	assert.Equal(t, pair.AccessExpiresAt, got.AccessExpiresAt)
	assert.Equal(t, pair.RefreshExpiresAt, got.RefreshExpiresAt)

	_, err = uc.ConsumeExchangeCode(context.Background(), code)
	assert.ErrorIs(t, err, apperr.ErrTokenNotFound)
}

func (d *oauthTestDeps) createUseCase() *OAuthUseCase {
	return NewOAuthUseCase(OAuthDeps{
		UserRepo: d.UserRepo, OAuthRepo: d.OAuthRepo, TM: d.TM,
		SettingsRepo: d.SettingsRepo, JWTService: d.JWTService,
		ProviderGateway: d.Gateway,
		CompParamUC:     d.CompParamUC,
		Cfg: OAuthConfig{
			StateSecret: "test-secret-for-oauth-state-1234",
			GitHub: OAuthProviderConfig{
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
	assert.ErrorIs(t, err, apperr.ErrOAuthUnsupportedProvider)
}

func TestOAuthUseCase_GetAuthURL_ProviderDisabled(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(defaultOAuthSettings(), nil)

	_, _, err := d.createUseCase().GetAuthURL(context.Background(), "google")
	assert.ErrorIs(t, err, apperr.ErrOAuthProviderDisabled)
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

	pair, err := uc.loginExistingOAuthUser(context.Background(), existingAcc, &OAuthProviderToken{AccessToken: "new-token"}, "github")
	require.NoError(t, err)
	assert.Equal(t, "new-access", pair.AccessToken)
}

func TestOAuthUseCase_LoginExistingUser_UserRepoError(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	existingAcc := newTestOAuthAccount(uuid.New(), "github", "gh-456", "token")

	d.UserRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("user not found"))

	_, err := uc.loginExistingOAuthUser(context.Background(), existingAcc, &OAuthProviderToken{AccessToken: "token"}, "github")
	assert.Error(t, err)
}

func TestOAuthUseCase_LoginExistingUser_WasInBannedTeam_Rejected(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	userID := uuid.New()
	existingAcc := newTestOAuthAccount(userID, "github", "gh-123", "old-token")
	existingUser := &domain.User{ID: userID, Email: "user@gh.com", Username: "ghuser", Role: domain.RoleUser, WasInBannedTeam: true}

	d.UserRepo.EXPECT().GetByID(mock.Anything, userID).Return(existingUser, nil)

	_, err := uc.loginExistingOAuthUser(context.Background(), existingAcc, &OAuthProviderToken{AccessToken: "new-token"}, "github")
	assert.ErrorIs(t, err, apperr.ErrInvalidCredentials)
}

func TestOAuthUseCase_RegisterNewUser_Success(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	profile := &domain.OAuthUserProfile{ID: "gh-789", Email: "newuser@gh.com", Username: "newghuser"}
	tokenPair := &jwtkit.TokenPair{AccessToken: "access", RefreshToken: "refresh", AccessExpiresAt: time.Now().Unix()}

	d.TM.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.UserRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Times(3)
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: true}, nil).Once()
	d.UserRepo.EXPECT().GetByEmail(mock.Anything, "newuser@gh.com").Return(nil, apperr.ErrUserNotFound)
	d.UserRepo.EXPECT().GetByUsername(mock.Anything, "newghuser").Return(nil, apperr.ErrUserNotFound)
	d.UserRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, u *domain.User) {
		u.ID = uuid.New()
	})
	d.OAuthRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.JWTService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, string(domain.RoleUser)).Return(tokenPair, nil)

	pair, err := uc.registerNewOAuthUser(context.Background(), profile, &OAuthProviderToken{AccessToken: "gh-access"}, "github")
	require.NoError(t, err)
	assert.Equal(t, "access", pair.AccessToken)
}

func TestOAuthUseCase_RegisterNewUser_TxError(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	profile := &domain.OAuthUserProfile{ID: "gh-err", Email: "err@gh.com", Username: "erruser"}

	d.TM.EXPECT().Run(mock.Anything, mock.Anything).Return(errors.New("tx error"))

	_, err := uc.registerNewOAuthUser(context.Background(), profile, &OAuthProviderToken{AccessToken: "token"}, "github")
	assert.Error(t, err)
}

func TestOAuthUseCase_RegisterNewUser_RegistrationClosed(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	uc := d.createUseCase()

	profile := &domain.OAuthUserProfile{ID: "gh-closed", Email: "new@gh.com", Username: "newuser"}

	d.TM.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: false}, nil).Once()

	_, err := uc.registerNewOAuthUser(context.Background(), profile, &OAuthProviderToken{AccessToken: "token"}, "github")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrRegistrationClosed)
}

func TestOAuthUseCase_RegisterNewUser_RegistrationVisibilityPrivate(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	d.CompParamUC = staticCompetitionParamUC{
		strings: map[string]string{"registration_visibility": "private"},
	}
	uc := d.createUseCase()

	profile := &domain.OAuthUserProfile{ID: "gh-private", Email: "private@gh.com", Username: "privateuser"}

	d.TM.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: true}, nil).Once()

	_, err := uc.registerNewOAuthUser(context.Background(), profile, &OAuthProviderToken{AccessToken: "token"}, "github")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrVisibilityForbidden)
}

func TestOAuthUseCase_RegisterNewUser_RegistrationCodeConfigured(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	d.CompParamUC = staticCompetitionParamUC{
		strings: map[string]string{
			"registration_visibility": "public",
			"registration_code":       "secret-sauce",
		},
	}
	uc := d.createUseCase()

	profile := &domain.OAuthUserProfile{ID: "gh-code", Email: "code@gh.com", Username: "codeuser"}

	d.TM.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: true}, nil).Once()

	_, err := uc.registerNewOAuthUser(context.Background(), profile, &OAuthProviderToken{AccessToken: "token"}, "github")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrRegistrationCodeRequired)
}

func TestOAuthUseCase_RegisterNewUser_MaxUsersReached(t *testing.T) {
	t.Parallel()
	d := newOAuthTestDeps(t)
	d.CompParamUC = staticCompetitionParamUC{
		strings: map[string]string{"registration_visibility": "public"},
	}
	uc := d.createUseCase()

	profile := &domain.OAuthUserProfile{ID: "gh-full", Email: "full@gh.com", Username: "fulluser"}

	d.TM.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.SettingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: true, MaxUsers: 1}, nil).Once()
	d.UserRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Once()
	d.UserRepo.EXPECT().CountActiveUsers(mock.Anything).Return(int64(1), nil).Once()

	_, err := uc.registerNewOAuthUser(context.Background(), profile, &OAuthProviderToken{AccessToken: "token"}, "github")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrMaxUsersReached)
}
