package webapi

import (
	"context"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// OAuthProviderAPI abstracts fetching a user profile from an OAuth provider.
type OAuthProviderAPI interface {
	FetchUserProfile(ctx context.Context, accessToken string) (*domain.OAuthUserProfile, error)
}
