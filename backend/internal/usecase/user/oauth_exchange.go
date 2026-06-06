package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

type storedTokenPair struct {
	AccessToken      string `json:"a"`
	RefreshToken     string `json:"r"`
	AccessExpiresAt  int64  `json:"ae"`
	RefreshExpiresAt int64  `json:"re"`
}

// IssueExchangeCode stores a token pair behind a short-lived one-time code so
// OAuth callbacks can redirect with a code instead of exposing tokens in URLs.
func (uc *OAuthUseCase) IssueExchangeCode(ctx context.Context, pair *usecase.TokenPair) (string, error) {
	if pair == nil {
		return "", fmt.Errorf("OAuthUseCase - IssueExchangeCode: token pair is nil")
	}

	if uc.deps.ExchangeStore == nil {
		return "", fmt.Errorf("OAuthUseCase - IssueExchangeCode: exchange store is not configured")
	}

	code, err := crypto.SecureRandomHex(oauthExchangeBytes)
	if err != nil {
		return "", fmt.Errorf("OAuthUseCase - IssueExchangeCode - SecureRandomHex: %w", err)
	}

	val, err := json.Marshal(storedTokenPair{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
	})
	if err != nil {
		return "", fmt.Errorf("OAuthUseCase - IssueExchangeCode - Marshal: %w", err)
	}

	if err := uc.deps.ExchangeStore.Set(ctx, oauthExchangePrefix+code, val, oauthExchangeTTL); err != nil {
		return "", fmt.Errorf("OAuthUseCase - IssueExchangeCode - ExchangeStore.Set: %w", err)
	}

	return code, nil
}

// ConsumeExchangeCode atomically consumes a one-time OAuth exchange code and
// returns the token pair stored by IssueExchangeCode.
func (uc *OAuthUseCase) ConsumeExchangeCode(ctx context.Context, code string) (*usecase.TokenPair, error) {
	if uc.deps.ExchangeStore == nil {
		return nil, fmt.Errorf("OAuthUseCase - ConsumeExchangeCode: exchange store is not configured")
	}

	val, err := uc.deps.ExchangeStore.GetDel(ctx, oauthExchangePrefix+code)
	if err != nil {
		if errors.Is(err, ErrOAuthExchangeCodeNotFound) {
			return nil, apperr.ErrTokenNotFound
		}

		return nil, fmt.Errorf("OAuthUseCase - ConsumeExchangeCode - ExchangeStore.GetDel: %w", err)
	}

	var stored storedTokenPair
	if err := json.Unmarshal(val, &stored); err != nil {
		return nil, fmt.Errorf("OAuthUseCase - ConsumeExchangeCode - Unmarshal: %w", err)
	}

	return &usecase.TokenPair{
		AccessToken:      stored.AccessToken,
		RefreshToken:     stored.RefreshToken,
		AccessExpiresAt:  stored.AccessExpiresAt,
		RefreshExpiresAt: stored.RefreshExpiresAt,
	}, nil
}
