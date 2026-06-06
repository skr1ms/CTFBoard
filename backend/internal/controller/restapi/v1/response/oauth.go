package response

import "github.com/TakuyaYagam1/AstroCTFb/internal/openapi"

func FromOAuthProviders(githubEnabled, googleEnabled bool) openapi.OAuthProvidersResponse {
	return openapi.OAuthProvidersResponse{
		Github: githubEnabled,
		Google: googleEnabled,
	}
}
