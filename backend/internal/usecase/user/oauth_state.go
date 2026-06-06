package user

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

// GetAuthURL builds the OAuth provider redirect URL with an HMAC-signed state parameter.
// The state is formatted as "<nonce_hex>.<hmac_sha256_hex>" where the nonce is
// cryptographically random; ValidateState verifies the signature on callback.
func (uc *OAuthUseCase) GetAuthURL(ctx context.Context, provider string) (authURL, state string, err error) {
	if err := uc.ensureProviderEnabled(ctx, provider); err != nil {
		return "", "", fmt.Errorf("OAuthUseCase - GetAuthURL - ensureProviderEnabled: %w", err)
	}

	nonceHex, err := crypto.SecureRandomHex(oauthNonceBytes)
	if err != nil {
		return "", "", fmt.Errorf("OAuthUseCase - GetAuthURL: %w", err)
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return "", "", fmt.Errorf("OAuthUseCase - GetAuthURL - hex.DecodeString: %w", err)
	}

	mac := hmac.New(sha256.New, uc.stateSecret)
	_, _ = mac.Write(nonce)
	sig := hex.EncodeToString(mac.Sum(nil))
	state = nonceHex + "." + sig

	authURL, err = uc.deps.ProviderGateway.AuthCodeURL(ctx, provider, state)
	if err != nil {
		return "", "", fmt.Errorf("OAuthUseCase - GetAuthURL - AuthCodeURL (%s): %w", provider, err)
	}

	return authURL, state, nil
}

// ValidateState performs a two-step verification: first a constant-time comparison
// of the cookie and query state strings to detect tampering, then it re-derives the
// HMAC from the embedded nonce and checks it against the signature in the state,
// preventing CSRF attacks via forged OAuth callbacks.
func (uc *OAuthUseCase) ValidateState(cookieState, queryState string) bool {
	if !hmac.Equal([]byte(cookieState), []byte(queryState)) {
		return false
	}

	parts := strings.SplitN(queryState, ".", oauthStatePartCount)
	if len(parts) != oauthStatePartCount {
		return false
	}

	nonce, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, uc.stateSecret)
	_, _ = mac.Write(nonce)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(parts[1]), []byte(expectedSig))
}
