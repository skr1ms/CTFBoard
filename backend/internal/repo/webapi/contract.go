package webapi

import "context"

// OAuthUserProfile holds the normalized profile data returned by an OAuth provider.
type OAuthUserProfile struct {
	ID       string
	Email    string
	Username string
}

// OAuthProviderAPI abstracts fetching a user profile from an OAuth provider.
type OAuthProviderAPI interface {
	FetchUserProfile(ctx context.Context, accessToken string) (*OAuthUserProfile, error)
}
