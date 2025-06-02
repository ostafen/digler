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
	"encoding/binary"
	"fmt"
	"io"
)

var sunAudioFileHeader = FileHeader{
	Ext: "au",
	Signatures: [][]byte{
		{0x2E, 0x73, 0x6E, 0x64},
	},
	ScanFile: ScanSunAudio,
}

const (
	// AU_MAGIC is the magic number for .au files: ".snd" in big-endian.
	AU_MAGIC uint32 = 0x2e736e64

	// MIN_AU_HEADER_SIZE is the minimum size of a valid AU header (6 * 4 bytes).
	MIN_AU_HEADER_SIZE = 24

	// AU_DATA_SIZE_UNKNOWN is the value used in the data_size field to indicate
	// that the data extends to the end of the file.
	AU_DATA_SIZE_UNKNOWN uint32 = 0xFFFFFFFF
)

// ScanSunAudio scans the input io.Reader strictly from the beginning
// for a valid AU file. It returns the total size of the detected
// AU data, or 0 and an error if no valid AU file is found at the beginning.
// The reader's position will be at the end of the AU data upon successful return.
func ScanSunAudio(r *Reader) (*ScanResult, error) {
	// We'll use a 24-byte buffer for the fixed part of the AU header.
	headerBuf := make([]byte, MIN_AU_HEADER_SIZE)

	// Read the first MIN_AU_HEADER_SIZE bytes
	n, err := io.ReadFull(r, headerBuf)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("reader too small (%d bytes) to contain a minimum AU header (%d bytes)", n, MIN_AU_HEADER_SIZE)
		}
		return nil, fmt.Errorf("failed to read AU header: %w", err)
	}

	// 1. Check Magic Number (Big Endian)
	magic := binary.BigEndian.Uint32(headerBuf[0:4])
	if magic != AU_MAGIC {
		return nil, fmt.Errorf("reader does not start with AU magic signature")
	}

	// 2. Read Header Size (Big Endian)
	headerSize := binary.BigEndian.Uint32(headerBuf[4:8])
	if headerSize < MIN_AU_HEADER_SIZE {
		return nil, fmt.Errorf("AU header size (%d) is invalid", headerSize)
	}

	// 3. Read Data Size (Big Endian)
	dataSize := binary.BigEndian.Uint32(headerBuf[8:12])

	bytesRead := uint64(MIN_AU_HEADER_SIZE)

	// If headerSize is greater than MIN_AU_HEADER_SIZE, we need to skip the
	// remaining part of the header.
	if headerSize > MIN_AU_HEADER_SIZE {
		skipBytes := int64(headerSize - MIN_AU_HEADER_SIZE)
		skipped, err := r.Discard(int(skipBytes))
		if err != nil {
			if err == io.EOF && skipped < int(skipBytes) {
				// Header is truncated, cannot be a valid AU file.
				return nil, fmt.Errorf("AU header truncated: expected %d bytes, got %d", headerSize, MIN_AU_HEADER_SIZE+uint32(skipped))
			}
			return nil, fmt.Errorf("failed to skip remaining header bytes: %w", err)
		}
		bytesRead += uint64(skipped)
	}

	var totalAUSize uint64
	if dataSize == AU_DATA_SIZE_UNKNOWN {
		// Data extends to the end of the file.
		// We've read the header, so the remaining size is what's left in the reader.
		// Since we don't know the full length, we'll return the bytes read so far.
		// The caller would typically read until EOF for the data.
		return &ScanResult{Size: bytesRead}, nil // Indicate that the valid part up to the header is found
	} else {
		// Data size is explicitly defined.
		totalAUSize = uint64(headerSize) + uint64(dataSize)

		// We need to advance the reader past the data chunk for the returned bytesRead to be accurate.
		// Calculate how many bytes of data are yet to be read.
		dataBytesToRead := int64(totalAUSize - bytesRead)

		if dataBytesToRead > 0 { // Only skip if there's data left to read
			skipped, err := io.CopyN(io.Discard, r, dataBytesToRead)
			if err != nil {
				if err == io.EOF && skipped < dataBytesToRead {
					// Data chunk is truncated. The valid AU ends here.
					return &ScanResult{Size: bytesRead + uint64(skipped)}, nil
				}
				return nil, fmt.Errorf("failed to skip AU data: %w", err)
			}
			bytesRead += uint64(skipped)
		}
	}

	// If we reach here, we've successfully scanned and potentially skipped all valid AU data.
	return &ScanResult{Size: totalAUSize}, nil
}
