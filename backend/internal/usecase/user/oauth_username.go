package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

const oauthUsernameHashLen = 12

func truncateUsername(s string) string {
	return truncateUsernameRunes(s, usernameMaxLen)
}

func truncateUsernameRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen])
}

func oauthUsernameHash(provider, providerID string) string {
	sum := sha256.Sum256([]byte(provider + ":" + providerID))

	return hex.EncodeToString(sum[:])[:oauthUsernameHashLen]
}

func oauthUsernameCandidates(desired, provider, providerID string) (string, string) {
	if desired == "" {
		desired = provider + "-user"
	}

	desired = truncateUsername(desired)
	suffix := fmt.Sprintf("-%s-%s", provider, oauthUsernameHash(provider, providerID))

	maxBaseLen := usernameMaxLen - len([]rune(suffix))
	if maxBaseLen < 1 {
		return desired, truncateUsername(provider + "-" + oauthUsernameHash(provider, providerID))
	}

	fallback := truncateUsernameRunes(desired, maxBaseLen) + suffix

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
