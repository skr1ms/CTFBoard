package wire

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/webapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
)

func ProvideOAuthHTTPClient() *http.Client {
	return webapi.NewOAuthHTTPClient()
}

func ProvideOAuthProviders(client *http.Client) map[string]webapi.OAuthProviderAPI {
	return map[string]webapi.OAuthProviderAPI{
		"github": webapi.NewGitHubAPI(client),
		"google": webapi.NewGoogleAPI(client),
	}
}

func ProvideOAuthGateway(client *http.Client, cfg user.OAuthConfig, providers map[string]webapi.OAuthProviderAPI) user.OAuthProviderGateway {
	return webapi.NewOAuthGateway(client, cfg, providers)
}

func ProvideOAuthConfig(cfg *config.Config) user.OAuthConfig {
	return user.OAuthConfig{
		StateSecret: cfg.StateSecret,
		GitHub: user.OAuthProviderConfig{
			ClientID:     cfg.GitHub.ClientID,
			ClientSecret: cfg.GitHub.ClientSecret,
			RedirectURL:  cfg.GitHub.RedirectURL,
		},
		Google: user.OAuthProviderConfig{
			ClientID:     cfg.Google.ClientID,
			ClientSecret: cfg.Google.ClientSecret,
			RedirectURL:  cfg.Google.RedirectURL,
		},
	}
}

type redisOAuthExchangeStore struct {
	client *redis.Client
}

func ProvideOAuthExchangeStore(redisClient *redis.Client) user.OAuthExchangeStore {
	return redisOAuthExchangeStore{client: redisClient}
}

func (s redisOAuthExchangeStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if s.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s redisOAuthExchangeStore) GetDel(ctx context.Context, key string) ([]byte, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	val, err := s.client.GetDel(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, user.ErrOAuthExchangeCodeNotFound
		}

		return nil, err
	}

	return val, nil
}

func ProvideOAuthUseCase(
	userRepo repo.UserRepository,
	oauthRepo repo.OAuthAccountRepository,
	TM repo.TransactionManager,
	settingsRepo repo.SettingsRepository,
	jwtService *jwtkit.JWTService,
	providerGateway user.OAuthProviderGateway,
	oauthCfg user.OAuthConfig,
	compRepo repo.CompetitionRepository,
	soloTeamCreator user.SoloTeamCreator,
	exchangeStore user.OAuthExchangeStore,
	l logkit.Logger,
) *user.OAuthUseCase {
	return user.NewOAuthUseCase(user.OAuthDeps{
		UserRepo:        userRepo,
		OAuthRepo:       oauthRepo,
		TM:              TM,
		SettingsRepo:    settingsRepo,
		JWTService:      jwtService,
		ProviderGateway: providerGateway,
		Cfg:             oauthCfg,
		CompRepo:        compRepo,
		SoloTeamCreator: soloTeamCreator,
		ExchangeStore:   exchangeStore,
		Logger:          l,
	})
}
