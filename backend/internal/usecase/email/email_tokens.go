package email

import (
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

// generateToken returns a cryptographically random hex string of the given byte
// length via crypto/rand. Used for verification and password-reset tokens.
func generateToken(length int) (string, error) {
	s, err := crypto.SecureRandomHex(length)
	if err != nil {
		return "", fmt.Errorf("EmailUseCase - generateToken: %w", err)
	}

	return s, nil
}

func hashToken(token string) string {
	return crypto.SHA256Hex(token)
}
