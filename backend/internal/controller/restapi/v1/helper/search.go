package helper

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const defaultSearchMaxLen = 100

func ValidateSearchQ(q string) bool {
	if utf8.RuneCountInString(q) > defaultSearchMaxLen {
		return false
	}
	for _, r := range q {
		if r == 0 || r == '\n' || r == '\r' || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func ValidateSearchQUsername(q string) bool {
	return ValidateSearchQ(q)
}

func EscapeILIKE(s string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = defaultSearchMaxLen
	}
	if utf8.RuneCountInString(s) > maxLen {
		runes := []rune(s)
		s = string(runes[:maxLen])
	}
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '%':
			b.WriteString(`\%`)
		case '_':
			b.WriteString(`\_`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func SanitizeSearchQ(q string, maxLen int) string {
	return EscapeILIKE(q, maxLen)
}
