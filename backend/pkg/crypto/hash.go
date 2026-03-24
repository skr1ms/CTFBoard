package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
)

func SHA256Hex(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func HashHex(h hash.Hash) string {
	return hex.EncodeToString(h.Sum(nil))
}

func SHA256Reader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("crypto.SHA256Reader: %w", err)
	}
	return HashHex(h), nil
}

func HMACSign(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func HMACVerify(key, data, mac []byte) bool {
	return hmac.Equal(mac, HMACSign(key, data))
}

func SecureRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto.SecureRandomHex: %w", err)
	}
	return hex.EncodeToString(b), nil
}
