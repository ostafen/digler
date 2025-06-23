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
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var mp3FileHeader = FileHeader{
	Ext:         "mp3",
	Description: "MPEG Audio Layer III audio format",
	Signatures: [][]byte{
		{0xFF, 0xFA},
		{0xFF, 0xFB},
		{0xFF, 0xF2},
		{0xFF, 0xF3},
		{0xFF, 0xE2},
		{0xFF, 0xE3},
		[]byte("ID3"),
	},
	ScanFile: ScanMP3,
}

// mp3Header represents the parsed information from a 4-byte MP3 frame header.
type mp3Header struct {
	MPEGVersion int  // 1, 2, or 2.5
	Layer       int  // 1, 2, or 3 (for MP3)
	Bitrate     int  // in kbps
	SampleRate  int  // in Hz
	Padding     bool // True if padding bit is set
	FrameSize   int  // calculated size of the entire frame in bytes
}

const (
	// MP3 frame size bounds (in bytes) used to validate potential MP3 frames during carving.
	// These bounds are based on MPEG-1 Layer III specifications:
	// - Minimum valid frame size ≈ 96 bytes (e.g., 32 kbps @ 48 kHz)
	// - Maximum valid frame size ≈ 1440 bytes (e.g., 320 kbps @ 32 kHz)
	// Frames outside this range are considered invalid to avoid false positives.
	minMp3FrameSize = 100
	maxMp3FrameSize = 1500
)

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
func skipID3v2Tag(r *Reader) (int, error) {
	headerBytes, err := r.Peek(10)
	if err != nil {
		return 0, err
	}

	// Check for "ID3" identifier
	if !bytes.Equal(headerBytes[0:3], []byte("ID3")) {
		return 0, nil // Not an ID3v2 tag
	}

	// Get tag size (bytes 6-9) - this is a synchsafe integer
	tagSize, err := parseSynchsafeInt(headerBytes[6:10])
	if err != nil {
		return 0, fmt.Errorf("failed to parse ID3v2 tag size: %w", err)
	}

	// Total bytes to skip: 10 bytes for the header + tagSize
	totalSkipBytes := 10 + tagSize

	_, err = r.Discard(totalSkipBytes)
	return totalSkipBytes, err
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
func ScanMP3(r *Reader) (*ScanResult, error) {
	var n int // Tracks the current position in the buffer

	// Check for and skip an initial ID3v2 tag (if present)
	// This is the only non-audio content allowed at the very start of the stream.
	skippedBytes, err := skipID3v2Tag(r)
	if err != nil {
		return nil, fmt.Errorf("error processing initial ID3v2 tag: %w", err)
	}
	n += skippedBytes // Move past the ID3v2 tag

	var headerBytes [4]byte
	numFrames := 0

	// currentOffset now points to the start of the first valid MP3 frame.
	// We will advance currentOffset frame by frame to find the end of the stream.
	for {
		// Re-parse header at currentOffset (this is the same as `header` for the first loop iteration)
		// This is critical for subsequent iterations to get the next frame's header.
		_, err = r.Read(headerBytes[:])
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF {
			break
		}

		header, ok := parseMP3Header(headerBytes[:])
		if !ok {
			// If the header is invalid, the contiguous stream ends here.
			break
		}

		if header.FrameSize < minMp3FrameSize || header.FrameSize > maxMp3FrameSize {
			return nil, fmt.Errorf("invalid mp3 frame size")
		}

		if _, err := r.Discard(header.FrameSize - 4); err != nil {
			return nil, err
		}

		n += header.FrameSize
		numFrames++
	}

	// A stream with only 1 frame is highly suspicious and often a false positive.
	// Requiring at least 2 frames provides more confidence.
	const MinimumRequiredFrames = 2
	if numFrames < MinimumRequiredFrames {
		return nil, fmt.Errorf("detected MP3 stream is too short (only %d frames)", numFrames)
	}
	return &ScanResult{Size: uint64(n)}, nil
}
