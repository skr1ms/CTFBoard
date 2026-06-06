package webapi

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
)

const (
	oauthProviderGitHub = "github"
	oauthProviderGoogle = "google"
	oauthProviderCount  = 2
)

type oauthProvider struct {
	config *oauth2.Config
	api    OAuthProviderAPI
}

// OAuthGateway adapts concrete OAuth providers to the usecase-owned gateway port.
type OAuthGateway struct {
	client    *http.Client
	providers map[string]oauthProvider
}

var _ user.OAuthProviderGateway = (*OAuthGateway)(nil)

func NewOAuthGateway(client *http.Client, cfg user.OAuthConfig, providers map[string]OAuthProviderAPI) *OAuthGateway {
	if client == nil {
		client = NewOAuthHTTPClient()
	}

	gateway := &OAuthGateway{
		client:    client,
		providers: make(map[string]oauthProvider, oauthProviderCount),
	}

	if cfg.GitHub.IsConfigured() {
		gateway.providers[oauthProviderGitHub] = oauthProvider{
			config: &oauth2.Config{
				ClientID:     cfg.GitHub.ClientID,
				ClientSecret: cfg.GitHub.ClientSecret,
				RedirectURL:  cfg.GitHub.RedirectURL,
				Scopes:       []string{"user:email", "read:user"},
				Endpoint:     github.Endpoint,
			},
			api: providers[oauthProviderGitHub],
		}
	}

	if cfg.Google.IsConfigured() {
		gateway.providers[oauthProviderGoogle] = oauthProvider{
			config: &oauth2.Config{
				ClientID:     cfg.Google.ClientID,
				ClientSecret: cfg.Google.ClientSecret,
				RedirectURL:  cfg.Google.RedirectURL,
				Scopes:       []string{"openid", "email", "profile"},
				Endpoint:     google.Endpoint,
			},
			api: providers[oauthProviderGoogle],
		}
	}

	return gateway
}

func (g *OAuthGateway) IsConfigured(provider string) bool {
	p, ok := g.providers[provider]

	return ok && p.config != nil
}

func (g *OAuthGateway) AuthCodeURL(_ context.Context, provider, state string) (string, error) {
	p, err := g.lookup(provider)
	if err != nil {
		return "", err
	}

	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (g *OAuthGateway) Exchange(ctx context.Context, provider, code string) (*user.OAuthProviderToken, error) {
	p, err := g.lookup(provider)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, g.client)

	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("OAuthGateway - Exchange (%s): %w", provider, err)
	}

	return &user.OAuthProviderToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
	}, nil
}

func (g *OAuthGateway) FetchUserProfile(ctx context.Context, provider, accessToken string) (*domain.OAuthUserProfile, error) {
	p, err := g.lookup(provider)
	if err != nil {
		return nil, err
	}

	if p.api == nil {
		return nil, fmt.Errorf("OAuthGateway - FetchUserProfile: provider API %q is not configured", provider)
	}

	profile, err := p.api.FetchUserProfile(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("OAuthGateway - FetchUserProfile (%s): %w", provider, err)
	}

	return profile, nil
}

func (g *OAuthGateway) lookup(provider string) (oauthProvider, error) {
	switch provider {
	case oauthProviderGitHub, oauthProviderGoogle:
	default:
		return oauthProvider{}, apperr.ErrOAuthUnsupportedProvider
	}

	p, ok := g.providers[provider]
	if !ok || p.config == nil {
		return oauthProvider{}, apperr.ErrOAuthProviderDisabled
	}

	return p, nil
}
