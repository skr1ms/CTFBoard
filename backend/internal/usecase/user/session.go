package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// RefreshTokenPair issues a new token pair from a refresh token while preserving
// only public authentication states at the usecase boundary.
func (uc *UserUseCase) RefreshTokenPair(ctx context.Context, refreshToken string) (*usecase.TokenPair, error) {
	pair, err := uc.deps.JWTService.RefreshTokens(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, apperr.ErrUserBanned) {
			return nil, fmt.Errorf("UserUseCase - RefreshTokenPair - JWTService.RefreshTokens: %w", apperr.ErrUserBanned)
		}

		return nil, fmt.Errorf("UserUseCase - RefreshTokenPair - JWTService.RefreshTokens: %w", apperr.ErrNotAuthenticated)
	}

	return tokenPairFromJWT(pair), nil
}

// Logout revokes the refresh token and best-effort revokes the current access
// token when one was supplied by the transport layer.
func (uc *UserUseCase) Logout(ctx context.Context, refreshToken string, accessToken *string) error {
	if accessToken != nil {
		token := strings.TrimSpace(*accessToken)
		if token != "" {
			if err := uc.deps.JWTService.RevokeAccessToken(ctx, token); err != nil {
				uc.deps.Logger.WithError(err).Warn("UserUseCase - Logout - JWTService.RevokeAccessToken")
			}
		}
	}

	if err := uc.deps.JWTService.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return fmt.Errorf("UserUseCase - Logout - JWTService.RevokeRefreshToken: %w", apperr.ErrNotAuthenticated)
	}

	return nil
}
