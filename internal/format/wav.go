package format

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// The size of a standard 16-byte WAV 'fmt ' sub-chunk.
// This is typical for PCM audio.
const (
	minWAVHeaderSize = 44 // Minimum size for a basic WAV header (RIFF, WAVE, fmt, data chunk headers)
	fmtChunkSizePCM  = 16 // Expected size for PCM audio in the 'fmt ' chunk
)

// ScanWAV scans the input byte buffer strictly from the beginning
// for a valid WAV file. It returns the end offset (exclusive) of the
// detected WAV data within the buffer, or 0 and an error if no
// valid WAV file is found at the beginning.
func ScanWAV(buf []byte) (uint64, error) {
	bufLen := len(buf)

	if bufLen < minWAVHeaderSize {
		return 0, fmt.Errorf("buffer too small (%d bytes) to contain a minimum WAV header (%d bytes)", bufLen, minWAVHeaderSize)
	}

	// 1. Check RIFF chunk
	// Offset 0-3: ChunkID "RIFF"
	if !bytes.Equal(buf[0:4], []byte("RIFF")) {
		return 0, fmt.Errorf("buffer does not start with RIFF signature")
	}

	// Offset 4-7: ChunkSize (Total file size - 8 bytes)
	// This is Little Endian
	riffChunkSize := binary.LittleEndian.Uint32(buf[4:8])
	if riffChunkSize+8 > uint32(bufLen) {
		return 0, fmt.Errorf("declared RIFF ChunkSize (%d) extends beyond buffer length", riffChunkSize)
	}
	// Note: We don't necessarily need to use riffChunkSize for the carve end
	// because a file could be truncated. We will determine the end based on
	// the actual 'data' chunk size.

	// Offset 8-11: Format "WAVE"
	if !bytes.Equal(buf[8:12], []byte("WAVE")) {
		return 0, fmt.Errorf("missing WAVE format identifier")
	}

	// 2. Find and parse 'fmt ' sub-chunk
	// This chunk should typically follow the 'WAVE' format, but its exact offset
	// can vary due to optional chunks before it. We'll scan for it.
	fmtChunkOffset := -1
	currentScanOffset := 12 // Start scanning after "WAVE"

	for currentScanOffset+8 <= bufLen { // Enough space for SubchunkID and SubchunkSize
		chunkID := buf[currentScanOffset : currentScanOffset+4]
		chunkSize := binary.LittleEndian.Uint32(buf[currentScanOffset+4 : currentScanOffset+8])

		if bytes.Equal(chunkID, []byte("fmt ")) {
			fmtChunkOffset = currentScanOffset
			if chunkSize != fmtChunkSizePCM {
				return 0, fmt.Errorf("unsupported 'fmt ' chunk size (%d), expected %d for PCM", chunkSize, fmtChunkSizePCM)
			}
			if currentScanOffset+8+int(chunkSize) > bufLen {
				return 0, fmt.Errorf("'fmt ' chunk declared size (%d) extends beyond buffer length", chunkSize)
			}
			break // Found 'fmt ' chunk
		}

		// Move to the next chunk
		currentScanOffset += 8 + int(chunkSize) // Advance past current chunk's header and data
		if currentScanOffset >= bufLen {
			break // Reached end of buffer while searching for 'fmt '
		}
	}

	if fmtChunkOffset == -1 {
		return 0, fmt.Errorf("missing 'fmt ' sub-chunk")
	}

	// At this point, we've located and validated the 'fmt ' chunk.
	// The standard WAV header is at least 44 bytes.
	// We proceed from the end of the 'fmt ' chunk.
	currentScanOffset = fmtChunkOffset + 8 + fmtChunkSizePCM // Move past fmt chunk header and data

	// 3. Find and parse 'data' sub-chunk
	dataChunkOffset := -1
	dataChunkSize := uint32(0)

	for currentScanOffset+8 <= bufLen { // Enough space for SubchunkID and SubchunkSize
		chunkID := buf[currentScanOffset : currentScanOffset+4]
		chunkSize := binary.LittleEndian.Uint32(buf[currentScanOffset+4 : currentScanOffset+8])

		if bytes.Equal(chunkID, []byte("data")) {
			dataChunkOffset = currentScanOffset
			dataChunkSize = chunkSize
			break // Found 'data' chunk
		}

		// Move to the next chunk
		currentScanOffset += 8 + int(chunkSize) // Advance past current chunk's header and data
		if currentScanOffset >= bufLen {
			break // Reached end of buffer while searching for 'data'
		}
	}

	if dataChunkOffset == -1 {
		return 0, fmt.Errorf("missing 'data' sub-chunk")
	}

	// Calculate the end offset of the actual audio data.
	// This is the start of the 'data' chunk + 8 bytes for its header + its declared size.
	audioDataEndOffset := dataChunkOffset + 8 + int(dataChunkSize)

	// Ensure the declared data chunk size does not exceed the buffer length.
	if audioDataEndOffset > bufLen {
		// If truncated, the valid data ends at bufLen
		return uint64(bufLen), nil // Return what's available
	}

	return uint64(audioDataEndOffset), nil
}
