package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/webapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestOAuthUseCase_GetAuthURL_Success(t *testing.T) {
	t.Parallel()
	h := NewOAuthTestHelper(t)
	h.SettingsRepo.EXPECT().Get(mock.Anything).Return(h.DefaultSettings(), nil)

	authURL, state, err := h.CreateUseCase().GetAuthURL(context.Background(), "github")
	require.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.NotEmpty(t, state)
	assert.Contains(t, authURL, "github.com")
}

func TestOAuthUseCase_GetAuthURL_UnsupportedProvider(t *testing.T) {
	t.Parallel()
	h := NewOAuthTestHelper(t)
	h.SettingsRepo.EXPECT().Get(mock.Anything).Return(h.DefaultSettings(), nil)

	_, _, err := h.CreateUseCase().GetAuthURL(context.Background(), "facebook")
	assert.ErrorIs(t, err, httperr.ErrOAuthUnsupportedProvider)
}

func TestOAuthUseCase_GetAuthURL_ProviderDisabled(t *testing.T) {
	t.Parallel()
	h := NewOAuthTestHelper(t)
	h.SettingsRepo.EXPECT().Get(mock.Anything).Return(h.DefaultSettings(), nil)

	_, _, err := h.CreateUseCase().GetAuthURL(context.Background(), "google")
	assert.ErrorIs(t, err, httperr.ErrOAuthProviderDisabled)
}

func TestOAuthUseCase_ValidateState_Match(t *testing.T) {
	t.Parallel()
	h := NewOAuthTestHelper(t)
	h.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{OAuthGithubEnabled: true}, nil)

	uc := h.CreateUseCase()
	_, state, err := uc.GetAuthURL(context.Background(), "github")
	require.NoError(t, err)

	assert.True(t, uc.ValidateState(state, state))
}

func TestOAuthUseCase_ValidateState_Mismatch(t *testing.T) {
	t.Parallel()
	h := NewOAuthTestHelper(t)

	assert.False(t, h.CreateUseCase().ValidateState("cookie-state", "different-state"))
}

func TestOAuthUseCase_LoginExistingUser_Success(t *testing.T) {
	t.Parallel()
	h := NewOAuthTestHelper(t)
	uc := h.CreateUseCase()

	userID := uuid.New()
	existingAcc := h.NewOAuthAccount(userID, "github", "gh-123", "old-token")
	existingUser := &entity.User{ID: userID, Email: "user@gh.com", Username: "ghuser", Role: entity.RoleUser}
	tokenPair := &jwt.TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh", AccessExpiresAt: time.Now().Unix()}

	h.OAuthRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	h.UserRepo.EXPECT().GetByID(mock.Anything, userID).Return(existingUser, nil)
	h.JWTService.EXPECT().GenerateTokenPair(userID, existingUser.Email, existingUser.Username, existingUser.Role).Return(tokenPair, nil)

	pair, err := uc.loginExistingOAuthUser(context.Background(), existingAcc, &oauth2.Token{AccessToken: "new-token"}, "github")
	require.NoError(t, err)
	assert.Equal(t, "new-access", pair.AccessToken)
}

func TestOAuthUseCase_LoginExistingUser_UserRepoError(t *testing.T) {
	t.Parallel()
	h := NewOAuthTestHelper(t)
	uc := h.CreateUseCase()

	existingAcc := h.NewOAuthAccount(uuid.New(), "github", "gh-456", "token")

	h.OAuthRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	h.UserRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("user not found"))

	_, err := uc.loginExistingOAuthUser(context.Background(), existingAcc, &oauth2.Token{AccessToken: "token"}, "github")
	assert.Error(t, err)
}

func TestOAuthUseCase_RegisterNewUser_Success(t *testing.T) {
	t.Parallel()
	h := NewOAuthTestHelper(t)
	uc := h.CreateUseCase()

	profile := &webapi.OAuthUserProfile{ID: "gh-789", Email: "newuser@gh.com", Username: "newghuser"}
	tokenPair := &jwt.TokenPair{AccessToken: "access", RefreshToken: "refresh", AccessExpiresAt: time.Now().Unix()}

	h.TM.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	h.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{RegistrationOpen: true}, nil).Once()
	h.UserRepo.EXPECT().GetByEmail(mock.Anything, "newuser@gh.com").Return(nil, httperr.ErrUserNotFound)
	h.UserRepo.EXPECT().GetByUsername(mock.Anything, "newghuser").Return(nil, httperr.ErrUserNotFound)
	h.UserRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, u *entity.User) {
		u.ID = uuid.New()
	})
	h.OAuthRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	h.JWTService.EXPECT().GenerateTokenPair(mock.Anything, "newuser@gh.com", "newghuser", entity.RoleUser).Return(tokenPair, nil)

	pair, err := uc.registerNewOAuthUser(context.Background(), profile, &oauth2.Token{AccessToken: "gh-access"}, "github")
	require.NoError(t, err)
	assert.Equal(t, "access", pair.AccessToken)
}

func TestOAuthUseCase_RegisterNewUser_TxError(t *testing.T) {
	t.Parallel()
	h := NewOAuthTestHelper(t)
	uc := h.CreateUseCase()

	profile := &webapi.OAuthUserProfile{ID: "gh-err", Email: "err@gh.com", Username: "erruser"}

	h.TM.EXPECT().Run(mock.Anything, mock.Anything).Return(errors.New("tx error"))

	_, err := uc.registerNewOAuthUser(context.Background(), profile, &oauth2.Token{AccessToken: "token"}, "github")
	assert.Error(t, err)
}

func TestOAuthUseCase_RegisterNewUser_RegistrationClosed(t *testing.T) {
	t.Parallel()
	h := NewOAuthTestHelper(t)
	uc := h.CreateUseCase()

	profile := &webapi.OAuthUserProfile{ID: "gh-closed", Email: "new@gh.com", Username: "newuser"}

	h.TM.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	h.SettingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{RegistrationOpen: false}, nil).Once()

	_, err := uc.registerNewOAuthUser(context.Background(), profile, &oauth2.Token{AccessToken: "token"}, "github")
	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrRegistrationClosed)
}
