package format

import "fmt"

// Helper to format bytes into human-readable units
// Helper to format bytes into human-readable units, avoiding .00 for whole numbers
func FormatBytes(b int64) string {
	const (
		_  = iota // ignore first value
		KB = 1 << (10 * iota)
		MB
		GB
		TB
	)

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
