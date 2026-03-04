package user

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/webapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/google/uuid"
)

type OAuthTestHelper struct {
	t            *testing.T
	UserRepo     *mocks.MockUserRepository
	OAuthRepo    *mocks.MockOAuthAccountRepository
	TM           *mocks.MockTransactionManager
	JWTService   *mocks.MockJWTService
	SettingsRepo *mocks.MockSettingsRepository
	ProviderAPI  *mocks.MockOAuthProviderAPI
}

func NewOAuthTestHelper(t *testing.T) *OAuthTestHelper {
	t.Helper()
	return &OAuthTestHelper{
		t:            t,
		UserRepo:     mocks.NewMockUserRepository(t),
		OAuthRepo:    mocks.NewMockOAuthAccountRepository(t),
		TM:           mocks.NewMockTransactionManager(t),
		JWTService:   mocks.NewMockJWTService(t),
		SettingsRepo: mocks.NewMockSettingsRepository(t),
		ProviderAPI:  mocks.NewMockOAuthProviderAPI(t),
	}
}

func (h *OAuthTestHelper) CreateUseCase() *OAuthUseCase {
	h.t.Helper()
	return NewOAuthUseCase(h.Deps())
}

func (h *OAuthTestHelper) Deps() OAuthDeps {
	h.t.Helper()
	return OAuthDeps{
		UserRepo:     h.UserRepo,
		OAuthRepo:    h.OAuthRepo,
		TM:           h.TM,
		SettingsRepo: h.SettingsRepo,
		JWTService:   h.JWTService,
		Providers: map[string]webapi.OAuthProviderAPI{
			"github": h.ProviderAPI,
		},
		Cfg: config.OAuth{
			StateSecret: "test-secret-for-oauth-state-1234",
			GitHub: config.OAuthProvider{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "http://localhost:3000/auth/github/callback",
			},
		},
	}
}

func (h *OAuthTestHelper) DefaultSettings() *entity.Settings {
	h.t.Helper()
	return &entity.Settings{
		OAuthGithubEnabled: true,
		OAuthGoogleEnabled: false,
	}
}

func (h *OAuthTestHelper) NewOAuthAccount(userID uuid.UUID, provider, providerUserID, accessToken string) *entity.OAuthAccount {
	h.t.Helper()
	return &entity.OAuthAccount{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		AccessToken:    accessToken,
	}
}

func (h *OAuthTestHelper) NewTokenPair(accessToken, refreshToken string) *jwt.TokenPair {
	h.t.Helper()
	return &jwt.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}
