package emailaddr

import "strings"

// Normalize applies the platform-wide email canonicalization used before
// lookup, uniqueness checks, and per-email rate-limit keys.
func Normalize(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
