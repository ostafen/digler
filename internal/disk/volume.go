// Copyright (c) 2025 Stefano Scafiti
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
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
