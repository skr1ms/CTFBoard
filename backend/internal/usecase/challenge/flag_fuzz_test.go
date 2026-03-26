package challenge

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func FuzzValidateFlagFormatRegex(f *testing.F) {
	f.Add("")
	f.Add("CTF{.*}")
	f.Add("[a-z]+")
	f.Add("(")
	f.Add("[")
	f.Add("(?P<name>.*)")
	f.Add("(?i)flag.*")
	f.Add(`\d{4}-\d{2}`)
	f.Add(strings.Repeat("a?", 100))
	f.Add("(?:a|b){0,100}")

	f.Fuzz(func(_ *testing.T, pattern string) {
		_ = validateFlagFormatRegex(&pattern) //nolint:errcheck // fuzz: intentionally ignoring error
	})
}

func FuzzFlagNormalise(f *testing.F) {
	f.Add("CTF{hello}")
	f.Add("   FLAG{trailing spaces}   ")
	f.Add("")
	f.Add("\x00\xff\n\r\t")
	f.Add(strings.Repeat("a", 4096))
	f.Add("旗{unicode_flag}")

	f.Fuzz(func(t *testing.T, input string) {
		normalised := strings.TrimSpace(input)
		normalised = strings.ToLower(normalised)
		hash := sha256.Sum256([]byte(normalised))

		result := hex.EncodeToString(hash[:])
		if len(result) != 64 {
			t.Errorf("sha256 hex must be 64 chars, got %d for input %q", len(result), input)
		}
	})
}
