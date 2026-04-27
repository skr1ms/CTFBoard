package crypto

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// NormalizeFlagInput strips invisible zero-width / directional / BOM Unicode characters
// and normalizes the string to NFC canonical form. Must be applied identically on both
// the flag creation/update path (hash side) and the submission path (input side) so that
// visually identical flags always compare equal regardless of how they were typed or pasted.
func NormalizeFlagInput(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\u200B', // ZERO WIDTH SPACE
			'\u200C', // ZERO WIDTH NON-JOINER
			'\u200D', // ZERO WIDTH JOINER
			'\u2060', // WORD JOINER
			'\uFEFF', // ZERO WIDTH NO-BREAK SPACE / BOM
			'\u202A', // LEFT-TO-RIGHT EMBEDDING
			'\u202B', // RIGHT-TO-LEFT EMBEDDING
			'\u202C', // POP DIRECTIONAL FORMATTING
			'\u202D', // LEFT-TO-RIGHT OVERRIDE
			'\u202E', // RIGHT-TO-LEFT OVERRIDE
			'\u200E', // LEFT-TO-RIGHT MARK
			'\u200F': // RIGHT-TO-LEFT MARK
			return -1
		}

		return r
	}, s)

	return norm.NFC.String(s)
}
