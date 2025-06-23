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
package format

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const (
	_  = iota // ignore first value
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

// Helper to format bytes into human-readable units, avoiding .00 for whole numbers
func FormatBytes(b int64) string {
	val := float64(b)
	var unit string

	switch {
	case b >= TB:
		val /= float64(TB)
		unit = "TB"
	case b >= GB:
		val /= float64(GB)
		unit = "GB"
	case b >= MB:
		val /= float64(MB)
		unit = "MB"
	case b >= KB:
		val /= float64(KB)
		unit = "KB"
	default:
		return fmt.Sprintf("%dB", b)
	}

	// Use %.0f for whole numbers, %.2f for numbers with decimals
	if val == float64(int(val)) {
		return fmt.Sprintf("%.0f%s", val, unit)
	}
	return fmt.Sprintf("%.2f%s", val, unit)
}

// ParseBytes converts a human-readable byte size string (e.g., "10MB", "1.5GB", "2048")
// into its corresponding int64 value in bytes. If no unit is specified, bytes are assumed.
func ParseBytes(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty input")
	}

	var i int
	for i = 0; i < len(s); i++ {
		if !unicode.IsDigit(rune(s[i])) && s[i] != '.' {
			break
		}
	}

	numStr := s[:i]
	unitStr := strings.ToUpper(strings.TrimSpace(s[i:]))

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %v", err)
	}

	var multiplier int64 = 1
	switch unitStr {
	case "", "B":
		multiplier = 1
	case "KB":
		multiplier = KB
	case "MB":
		multiplier = MB
	case "GB":
		multiplier = GB
	case "TB":
		multiplier = TB
	default:
		return 0, fmt.Errorf("unknown unit: %s", unitStr)
	}
	return uint64(num * float64(multiplier)), nil
}
