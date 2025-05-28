package format

import (
	"encoding/binary"
	"fmt"
)

const (
	// AU_MAGIC is the magic number for .au files: ".snd" in big-endian.
	AU_MAGIC uint32 = 0x2e736e64

	// MIN_AU_HEADER_SIZE is the minimum size of a valid AU header (6 * 4 bytes).
	MIN_AU_HEADER_SIZE = 24

	// AU_DATA_SIZE_UNKNOWN is the value used in the data_size field to indicate
	// that the data extends to the end of the file.
	AU_DATA_SIZE_UNKNOWN uint32 = 0xFFFFFFFF
)

// ScanSunAudio scans the input byte buffer strictly from the beginning
// for a valid AU file. It returns the end offset (exclusive) of the
// detected AU data within the buffer, or 0 and an error if no
// valid AU file is found at the beginning.
func ScanSunAudio(buf []byte) (uint64, error) {
	bufLen := len(buf)

	if bufLen < MIN_AU_HEADER_SIZE {
		return 0, fmt.Errorf("buffer too small (%d bytes) to contain a minimum AU header (%d bytes)", bufLen, MIN_AU_HEADER_SIZE)
	}

	// 1. Check Magic Number (Big Endian)
	magic := binary.BigEndian.Uint32(buf[0:4])
	if magic != AU_MAGIC {
		return 0, fmt.Errorf("buffer does not start with AU magic signature")
	}

	// 2. Read Header Size (Big Endian)
	headerSize := binary.BigEndian.Uint32(buf[4:8])
	if headerSize < MIN_AU_HEADER_SIZE {
		return 0, fmt.Errorf("AU header size (%d) is invalid", headerSize)
	}
	if headerSize > uint32(bufLen) {
		return 0, fmt.Errorf("AU header size (%d) extends beyond buffer length", headerSize)
	}
	// Header size must be a multiple of 2 (even)
	if headerSize%2 != 0 {
		return 0, fmt.Errorf("AU header size (%d) is not an even number", headerSize)
	}

	// 3. Read Data Size (Big Endian)
	dataSize := binary.BigEndian.Uint32(buf[8:12])

	var endOffset int
	if dataSize == AU_DATA_SIZE_UNKNOWN {
		// Data extends to the end of the buffer
		endOffset = bufLen
	} else {
		// Data size is explicitly defined
		endOffset = int(headerSize + dataSize)

		// Check if the declared data size extends beyond the buffer's actual length
		if endOffset > bufLen {
			// If truncated, the valid data ends at bufLen
			return uint64(bufLen), nil // Return what's available
		}
	}
	return uint64(endOffset), nil
}
