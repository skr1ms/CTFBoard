package challenge

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
// each resulting extension against forbiddenUploadExtensions. Splitting on all
// dots (not just the last) blocks double-extension attacks such as "shell.php.txt"
// where the final extension appears safe but an intermediate one is dangerous.
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

var dangerousMagicBytes = [][]byte{
	[]byte("MZ"),
	[]byte("\x7fELF"),
	[]byte("#!"),
	[]byte("%PDF"),
	[]byte("\xd0\xcf\x11\xe0"),
}

// ValidateFileMagic checks the first bytes of the uploaded content against known
// executable/dangerous magic byte sequences (MZ/PE, ELF, shebang, PDF, OLE).
// Returning false blocks the upload regardless of the declared file extension,
// preventing extension-spoofing attacks where a binary is renamed to ".pdf".
// PDF (%PDF) and OLE (0xD0CF11E0) are blocked because they are common exploit carriers.
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
