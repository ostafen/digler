package disk

import (
	"runtime"
	"strings"
	"unicode"
)

// NormalizeVolumePath checks if a given path is a Windows volume path
// and normalizes it to \\.\C: format if running on Windows.
// Otherwise, returns the path unchanged.
func NormalizeVolumePath(path string) string {
	if runtime.GOOS != "windows" {
		return path // Only normalize on Windows
	}

	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "/", `\`)
	upper := strings.ToUpper(path)

	// Already a raw volume path like \\.\C:
	if strings.HasPrefix(upper, `\\.\`) {
		return upper
	}

	// Handle paths like "C:" or "C:\" (must be drive letter only)
	if len(upper) >= 2 && upper[1] == ':' && unicode.IsLetter(rune(upper[0])) {
		// Normalize to \\.\C:
		return `\\.\` + strings.ToUpper(string(upper[0])) + `:`
	}

	return path // Not a volume path
}
