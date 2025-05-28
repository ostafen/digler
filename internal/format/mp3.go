package format

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// mp3Header represents the parsed information from a 4-byte MP3 frame header.
type mp3Header struct {
	MPEGVersion int  // 1, 2, or 2.5
	Layer       int  // 1, 2, or 3 (for MP3)
	Bitrate     int  // in kbps
	SampleRate  int  // in Hz
	Padding     bool // True if padding bit is set
	FrameSize   int  // calculated size of the entire frame in bytes
}

// --- Global Lookup Tables for MP3 Header Parsing ---
// These tables are derived from the MPEG Audio specification.

// Bitrate values in kbps for MPEG 1 Layer III (MP3)
var bitrateMPEG1Layer3 = []int{
	0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0,
}

// Bitrate values in kbps for MPEG 2/2.5 Layer III (MP3)
var bitrateMPEG2Layer3 = []int{
	0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0,
}

// Sample rate values in Hz for different MPEG versions.
// Indexed by MPEG version bits (0 for 2.5, 2 for 2, 3 for 1).
var sampleRateTable = [][]int{
	{11025, 12000, 8000, 0},  // Index 0: MPEG 2.5
	{0, 0, 0, 0},             // Index 1: Reserved
	{22050, 24000, 16000, 0}, // Index 2: MPEG 2
	{44100, 48000, 32000, 0}, // Index 3: MPEG 1
}

// parseSynchsafeInt parses a 4-byte synchsafe integer.
// In a synchsafe integer, the most significant bit of every byte is zero.
func parseSynchsafeInt(b []byte) (int, error) {
	if len(b) < 4 {
		return 0, fmt.Errorf("byte slice too short for synchsafe int (expected 4, got %d)", len(b))
	}
	val := int(b[0]&0x7F)<<21 |
		int(b[1]&0x7F)<<14 |
		int(b[2]&0x7F)<<7 |
		int(b[3]&0x7F)
	return val, nil
}

// skipID3v2Tag checks for an ID3v2 tag at the current buffer offset and calculates its size.
// It returns the number of bytes to skip (0 if no tag) and an error, if any.
// This function is intended for use ONLY at the very beginning of the buffer.
func skipID3v2Tag(data []byte, currentOffset int) (int, error) {
	if currentOffset+10 > len(data) {
		return 0, nil // Not enough data for ID3 header
	}

	headerBytes := data[currentOffset : currentOffset+10]

	// Check for "ID3" identifier
	if !bytes.Equal(headerBytes[0:3], []byte("ID3")) {
		return 0, nil // Not an ID3v2 tag
	}

	// Get tag size (bytes 6-9) - this is a synchsafe integer
	tagSize, err := parseSynchsafeInt(headerBytes[6:10])
	if err != nil {
		return 0, fmt.Errorf("failed to parse ID3v2 tag size at offset %d: %w", currentOffset, err)
	}

	// Total bytes to skip: 10 bytes for the header + tagSize
	totalSkipBytes := 10 + tagSize

	if currentOffset+totalSkipBytes > len(data) {
		return 0, fmt.Errorf("ID3v2 tag declared size (%d) extends beyond buffer end", totalSkipBytes)
	}

	return totalSkipBytes, nil
}

// parseMP3Header attempts to parse a 4-byte MP3 frame header.
// It returns an mp3Header struct and a boolean indicating success.
func parseMP3Header(headerBytes []byte) (mp3Header, bool) {
	if len(headerBytes) < 4 {
		return mp3Header{}, false
	}

	// Combine bytes into a 32-bit unsigned integer for easier bitwise operations
	header := binary.BigEndian.Uint32(headerBytes)

	// Check for sync word (first 11 bits must be 1s).
	// 0xFFE00000 is 11111111111000000000000000000000 in binary.
	if (header & 0xFFE00000) != 0xFFE00000 {
		return mp3Header{}, false
	}

	// Extract MPEG version (bits 12-13 from MSB, 0-indexed from 0xFFF)
	// 11 = MPEG 1, 10 = MPEG 2, 00 = MPEG 2.5, 01 = Reserved
	mpegVersionBits := int((header >> 19) & 0x03)
	var mpegVersion int
	switch mpegVersionBits {
	case 3:
		mpegVersion = 1
	case 2:
		mpegVersion = 2
	case 0:
		mpegVersion = 25 // Representing 2.5
	case 1:
		return mp3Header{}, false // Reserved version
	}

	// Extract Layer (bits 14-15 from MSB)
	// 11 = Layer I, 10 = Layer II, 01 = Layer III, 00 = Reserved
	layerBits := int((header >> 17) & 0x03)
	if layerBits != 1 { // We are specifically looking for MP3 (Layer III)
		return mp3Header{}, false
	}

	// Extract Bitrate Index (bits 16-19 from MSB)
	bitrateIndex := int((header >> 12) & 0x0F)
	var bitrate int
	if mpegVersion == 1 {
		if bitrateIndex == 0 || bitrateIndex == 15 { // 0 is 'free', 15 is 'bad'
			return mp3Header{}, false
		}
		bitrate = bitrateMPEG1Layer3[bitrateIndex]
	} else { // MPEG 2 or 2.5
		if bitrateIndex == 0 || bitrateIndex == 15 { // 0 is 'free', 15 is 'bad'
			return mp3Header{}, false
		}
		bitrate = bitrateMPEG2Layer3[bitrateIndex]
	}

	// Extract Sample Rate Index (bits 20-21 from MSB)
	sampleRateIndex := int((header >> 10) & 0x03)
	if sampleRateIndex == 3 { // Reserved sample rate
		return mp3Header{}, false
	}
	sampleRate := sampleRateTable[mpegVersionBits][sampleRateIndex]
	if sampleRate == 0 { // Should not happen if previous checks pass, but for safety
		return mp3Header{}, false
	}

	// Extract Padding bit (bit 22 from MSB)
	padding := ((header >> 9) & 0x01) != 0

	// Calculate Frame Size in bytes
	// FrameSize = ((1152 samples/frame * Bitrate(bits/s)) / SampleRate(Hz)) / 8 bits/byte + Padding
	// For Layer III, samples per frame is 1152. Bitrate is in kbps, so multiply by 1000 to get bps.
	frameSize := ((1152 * bitrate * 1000) / sampleRate) / 8
	if padding {
		frameSize += 1
	}

	if frameSize <= 4 { // A valid frame must be larger than its own 4-byte header
		return mp3Header{}, false
	}

	return mp3Header{
		MPEGVersion: mpegVersion,
		Layer:       3, // Always Layer III for MP3
		Bitrate:     bitrate,
		SampleRate:  sampleRate,
		Padding:     padding,
		FrameSize:   frameSize,
	}, true
}

// ScanMP3 scans the input byte slice, strictly from the beginning,
// for a contiguous MP3 stream. It returns the end offset (exclusive)
// of the detected stream within the buffer, or 0 and an error if no
// valid stream is found. An optional ID3v2 tag at the very beginning
// is allowed and skipped.
func ScanMP3(buf []byte) (uint64, error) {
	dataLen := len(buf)
	if dataLen < 4 { // We need at least 4 bytes to check for a header
		return 0, fmt.Errorf("input buffer too small to contain MP3 frames")
	}

	var currentOffset int // Tracks the current position in the buffer

	// 1. Check for and skip an initial ID3v2 tag (if present)
	// This is the only non-audio content allowed at the very start of the stream.
	skippedBytes, err := skipID3v2Tag(buf, 0)
	if err != nil {
		return 0, fmt.Errorf("error processing initial ID3v2 tag: %w", err)
	}
	currentOffset += skippedBytes // Move past the ID3v2 tag

	// 2. The very next 4 bytes *must* be a valid MP3 frame header
	if currentOffset+4 > dataLen {
		return 0, fmt.Errorf("buffer too short for MP3 header after initial tag check (current offset: %d)", currentOffset)
	}

	headerBytes := buf[currentOffset : currentOffset+4]
	header, ok := parseMP3Header(headerBytes)
	if !ok {
		return 0, fmt.Errorf("no valid MP3 header found at the expected start of the stream")
	}

	carvedFramesInStream := 0

	// 3. Continue parsing contiguous MP3 frames
	// currentOffset now points to the start of the first valid MP3 frame.
	// We will advance currentOffset frame by frame to find the end of the stream.
	for {
		// Ensure enough data for the current frame header
		if currentOffset+4 > dataLen {
			break // End of buffer, no more frames
		}

		// Re-parse header at currentOffset (this is the same as `header` for the first loop iteration)
		// This is critical for subsequent iterations to get the next frame's header.
		headerBytes = buf[currentOffset : currentOffset+4]
		header, ok = parseMP3Header(headerBytes)

		if !ok {
			// If the header is invalid, the contiguous stream ends here.
			break
		}

		// Check if the full frame extends beyond the buffer
		if currentOffset+header.FrameSize > dataLen {
			break // Frame extends past end of data
		}

		carvedFramesInStream++
		currentOffset += header.FrameSize // Advance to the start of the next potential frame
	}

	// 4. Final Validation and Return
	// A stream with only 1 frame is highly suspicious and often a false positive.
	// Requiring at least 2 frames provides more confidence.
	const MinimumRequiredFrames = 2
	if carvedFramesInStream < MinimumRequiredFrames {
		return 0, fmt.Errorf("detected MP3 stream is too short (only %d frames)", carvedFramesInStream)
	}
	return uint64(currentOffset), nil
}
