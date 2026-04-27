package crypto_test

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

func TestNormalizeFlagInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain ASCII is unchanged",
			input: "flag{hello_world}",
			want:  "flag{hello_world}",
		},
		{
			name:  "trailing ZWSP is stripped",
			input: "flag{abc}\u200B",
			want:  "flag{abc}",
		},
		{
			name:  "leading BOM is stripped",
			input: "\uFEFFflag{abc}",
			want:  "flag{abc}",
		},
		{
			name:  "embedded zero-width chars are stripped",
			input: "flag\u200B{\u200Cabc\u200D}",
			want:  "flag{abc}",
		},
		{
			name:  "directional marks are stripped",
			input: "\u200Eflag{abc}\u200F",
			want:  "flag{abc}",
		},
		{
			name:  "NFD normalized to NFC",
			input: "flag{cafe\u0301}", // e + combining acute accent (NFD)
			want:  "flag{caf\u00E9}",  // é (NFC)
		},
		{
			name:  "already NFC is unchanged",
			input: "flag{caf\u00E9}",
			want:  "flag{caf\u00E9}",
		},
		{
			name:  "normal ASCII whitespace is NOT stripped (trim is caller's job)",
			input: "flag{abc} ",
			want:  "flag{abc} ",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "ZWSP-only string becomes empty",
			input: "\u200B\uFEFF\u200C",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := crypto.NormalizeFlagInput(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeFlagInput(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestNormalizeFlagInputIdempotent ensures that applying the function twice gives the same
// result as applying it once.
func TestNormalizeFlagInputIdempotent(t *testing.T) {
	inputs := []string{
		"flag{hello}",
		"flag{abc}\u200B",
		"flag{cafe\u0301}",
		"\uFEFFflag{\u200Btest\u200D}",
	}
	for _, s := range inputs {
		once := crypto.NormalizeFlagInput(s)

		twice := crypto.NormalizeFlagInput(once)
		if once != twice {
			t.Errorf("NormalizeFlagInput is not idempotent for %q: once=%q twice=%q", s, once, twice)
		}
	}
}
