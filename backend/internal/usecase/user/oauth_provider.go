package user

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

// ensureProviderEnabled checks runtime settings and static provider config
// before the OAuth adapter performs provider-specific protocol work.
func (uc *OAuthUseCase) ensureProviderEnabled(ctx context.Context, provider string) error {
	settings, err := uc.deps.SettingsRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("OAuthUseCase - ensureProviderEnabled - GetSettings: %w", err)
	}

	if uc.deps.ProviderGateway == nil {
		return apperr.ErrOAuthProviderDisabled
	}

	switch provider {
	case "github":
		if !settings.OAuthGithubEnabled || !uc.deps.ProviderGateway.IsConfigured(provider) {
			return apperr.ErrOAuthProviderDisabled
		}

		return nil

	case "google":
		if !settings.OAuthGoogleEnabled || !uc.deps.ProviderGateway.IsConfigured(provider) {
			return apperr.ErrOAuthProviderDisabled
		}

		return nil

	default:
		return apperr.ErrOAuthUnsupportedProvider
	}
}
