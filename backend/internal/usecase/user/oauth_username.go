package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

func truncateUsername(s string) string {
	runes := []rune(s)
	if len(runes) <= usernameMaxLen {
		return s
	}

	return string(runes[:usernameMaxLen])
}

func oauthUsernameCandidates(desired, provider, providerID string) (string, string) {
	if desired == "" {
		desired = provider + "-user"
	}

	desired = truncateUsername(desired)
	fallback := truncateUsername(fmt.Sprintf("%s-%s-%s", desired, provider, providerID))

	return desired, fallback
}

// resolveUsername picks a username (within the current ctx transaction) to avoid TOCTOU
// between the uniqueness check and the subsequent Create call.
func (uc *OAuthUseCase) resolveUsername(ctx context.Context, desired, provider, providerID string) (string, error) {
	desired, fallback := oauthUsernameCandidates(desired, provider, providerID)

	_, err := uc.deps.UserRepo.GetByUsername(ctx, desired)
	if errors.Is(err, apperr.ErrUserNotFound) {
		return desired, nil
	}

	if err != nil {
		return "", fmt.Errorf("OAuthUseCase - resolveUsername - UserRepo.GetByUsername: %w", err)
	}

	// desired is taken - use provider-scoped fallback that includes the unique provider ID
	_, err = uc.deps.UserRepo.GetByUsername(ctx, fallback)
	if errors.Is(err, apperr.ErrUserNotFound) {
		return fallback, nil
	}

	if err != nil {
		return "", fmt.Errorf("OAuthUseCase - resolveUsername - UserRepo.GetByUsername fallback: %w", err)
	}

	return "", apperr.ErrUsernameTaken
}
