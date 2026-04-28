package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
)

// SHA256Hex returns the hex-encoded SHA-256 digest of the given string.
func SHA256Hex(data string) string {
	h := sha256.Sum256([]byte(data))

	return hex.EncodeToString(h[:])
}

// HashHex finalizes h and returns its hex-encoded digest.
func HashHex(h hash.Hash) string {
	return hex.EncodeToString(h.Sum(nil))
}

// HMACSign returns the HMAC-SHA256 signature of data using key.
func HMACSign(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data) //nolint:revive // hash.Write never returns error

	return mac.Sum(nil)
}

// HMACVerify reports whether mac is a valid HMAC-SHA256 signature of data under key.
func HMACVerify(key, data, mac []byte) bool {
	return hmac.Equal(mac, HMACSign(key, data))
}

// SecureRandomHex generates n cryptographically random bytes and returns them as a hex string of length 2n.
func SecureRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto.SecureRandomHex: %w", err)
	}

	return hex.EncodeToString(b), nil
}
