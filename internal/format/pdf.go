package format

import (
	"bytes"
	"errors"
	"strconv"
)

func ScanPDF(buf []byte) (uint64, error) {
	if len(buf) < 8 || !bytes.HasPrefix(buf, []byte("%PDF-1.")) {
		return 0, errors.New("not a valid PDF")
	}

	// 1. Look for "/Linearized" then "/L <size>"
	linearizedIndex := bytes.Index(buf, []byte("/Linearized"))
	if linearizedIndex != -1 {
		// Limit scan to 512 bytes after /Linearized
		end := linearizedIndex + 512
		if end > len(buf) {
			end = len(buf)
		}
		segment := buf[linearizedIndex:end]

		// Look for "/L "
		for i := 0; i < len(segment)-3; i++ {
			if segment[i] == '/' && segment[i+1] == 'L' && segment[i+2] == ' ' {
				// Parse number after "/L "
				j := i + 3
				for j < len(segment) && segment[j] == ' ' {
					j++
				}
				start := j
				for j < len(segment) && segment[j] >= '0' && segment[j] <= '9' {
					j++
				}
				numStr := string(segment[start:j])
				if size, err := strconv.ParseUint(numStr, 10, 64); err == nil {
					return size, nil
				}
			}
		}
	}

	// 2. Fallback: Look for last %EOF
	lastEOF := bytes.LastIndex(buf, []byte("%EOF"))
	if lastEOF != -1 {
		return uint64(lastEOF + len("%EOF")), nil
	}
	return 0, errors.New("could not determine PDF size")
}
