package user

import (
	"github.com/wahrwelt-kit/go-jwtkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func tokenPairFromJWT(pair *jwtkit.TokenPair) *usecase.TokenPair {
	if pair == nil {
		return nil
	}

	return &usecase.TokenPair{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
	}
}
