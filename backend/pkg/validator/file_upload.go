package validator

import (
	"bytes"
	"strings"
)

var forbiddenUploadExtensions = map[string]struct{}{
	".html": {}, ".htm": {}, ".xhtml": {}, ".svg": {}, ".xml": {},
	".exe": {}, ".bat": {}, ".cmd": {}, ".sh": {}, ".ps1": {},
	".php": {}, ".php3": {}, ".php4": {}, ".php5": {}, ".phtml": {},
	".js": {}, ".mjs": {}, ".cjs": {},
	".vbs": {}, ".wsf": {}, ".jse": {},
	".jar": {}, ".msi": {},
	".py": {}, ".rb": {}, ".pl": {},
}

// ValidateUploadFilename splits the lowercase filename on every "." and checks
// each extension segment. This blocks double-extension uploads such as
// "shell.php.txt" where the final extension appears safe.
func ValidateUploadFilename(filename string) bool {
	lower := strings.ToLower(filename)

	parts := strings.Split(lower, ".")
	for i := 1; i < len(parts); i++ {
		ext := "." + parts[i]
		if _, forbidden := forbiddenUploadExtensions[ext]; forbidden {
			return false
		}
	}

	return true
}

func ValidateChallengeUploadFilename(filename string) bool {
	filename = strings.TrimSpace(filename)
	if filename == "" || strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\`) {
		return false
	}

	for _, r := range filename {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}

	return true
}

var dangerousMagicBytes = [][]byte{
	[]byte("MZ"),
	[]byte("\x7fELF"),
	[]byte("#!"),
	[]byte("%PDF"),
	[]byte("\xd0\xcf\x11\xe0"),
}

// ValidateFileMagic checks the first bytes of uploaded content against known
// executable or high-risk magic byte signatures.
func ValidateFileMagic(header []byte) bool {
	if len(header) == 0 {
		return true
	}

	for _, magic := range dangerousMagicBytes {
		if len(header) >= len(magic) && bytes.Equal(header[:len(magic)], magic) {
			return false
		}
	}

	return true
}
