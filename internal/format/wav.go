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

var wavFileHeader = FileHeader{
	Ext: "wav",
	Signatures: [][]byte{
		[]byte("RIFF"),
		[]byte("RIFX"),
	},
	ScanFile: ScanWAV,
}

// The size of a standard 16-byte WAV 'fmt ' sub-chunk.
// This is typical for PCM audio.
const (
	minWAVHeaderSize = 44 // Minimum size for a basic WAV header (RIFF, WAVE, fmt, data chunk headers)
	fmtChunkSizePCM  = 16 // Expected size for PCM audio in the 'fmt ' chunk
)

// ScanWAV scans the input io.Reader strictly from the beginning
// for a valid WAV file. It returns the total size of the detected
// WAV data, or 0 and an error if no valid WAV file is found at the beginning.
// The reader's position will be at the end of the WAV data upon successful return.
func ScanWAV(r *Reader) (*ScanResult, error) {
	// We'll use a small buffer for reading headers and sizes.

	var headerBuf [8]byte // For ChunkID and ChunkSize

	// 1. Check RIFF chunk
	// Read Offset 0-3: ChunkID "RIFF" and Offset 4-7: ChunkSize
	n, err := io.ReadFull(r, headerBuf[:])
	if err != nil {
		return nil, fmt.Errorf("failed to read RIFF chunk header: %w", err)
	}
	if n < 8 {
		return nil, fmt.Errorf("reader too small (%d bytes) to contain RIFF chunk header", n)
	}

	if string(headerBuf[0:4]) != "RIFF" {
		return nil, fmt.Errorf("reader does not start with RIFF signature")
	}

	// riffChunkSize is the total file size minus 8 bytes (RIFF ChunkID and ChunkSize themselves).
	// This includes WAVE ID, fmt chunk, data chunk, and any other chunks.
	riffChunkSize := binary.LittleEndian.Uint32(headerBuf[4:8])

	// Read Offset 8-11: Format "WAVE"
	waveFormatBuf := make([]byte, 4)
	_, err = io.ReadFull(r, waveFormatBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAVE format identifier: %w", err)
	}
	if string(waveFormatBuf) != "WAVE" {
		return nil, fmt.Errorf("missing WAVE format identifier")
	}

	bytesRead := uint64(12) // RIFF (8 bytes) + WAVE (4 bytes)

	// 2. Find and parse 'fmt ' sub-chunk
	fmtChunkFound := false
	for bytesRead < uint64(riffChunkSize)+8 { // Ensure we don't read beyond the declared RIFF chunk
		n, err = io.ReadFull(r, headerBuf[:])
		if err != nil {
			if err == io.EOF && bytesRead+uint64(n) < uint64(riffChunkSize)+8 {
				// We hit EOF before fully parsing the RIFF chunk based on its declared size.
				// This indicates a truncated file, but we should still find fmt and data if possible.
				break
			}
			return nil, fmt.Errorf("failed to read chunk header while searching for 'fmt ': %w", err)
		}

		chunkID := string(headerBuf[0:4])
		chunkSize := binary.LittleEndian.Uint32(headerBuf[4:8])
		bytesRead += 8

		if chunkID == "fmt " {
			if chunkSize != fmtChunkSizePCM {
				return nil, fmt.Errorf("unsupported 'fmt ' chunk size (%d), expected %d for PCM", chunkSize, fmtChunkSizePCM)
			}

			// Read the fmt chunk data
			fmtDataBuf := make([]byte, chunkSize)
			n, err = io.ReadFull(r, fmtDataBuf)
			if err != nil {
				return nil, fmt.Errorf("failed to read 'fmt ' chunk data: %w", err)
			}
			bytesRead += uint64(n)
			fmtChunkFound = true
			break // Found 'fmt ' chunk, move to data
		}

		// Skip over the current chunk's data
		skipped, err := r.Discard(int(chunkSize))
		if err != nil {
			if err == io.EOF && skipped < int(chunkSize) {
				// Truncated chunk data, but we might still find 'data' if it's earlier.
				break
			}
			return nil, fmt.Errorf("failed to skip chunk data while searching for 'fmt ': %w", err)
		}
		bytesRead += uint64(skipped)
	}

	if !fmtChunkFound {
		return nil, fmt.Errorf("missing 'fmt ' sub-chunk")
	}

	// 3. Find and parse 'data' sub-chunk
	dataChunkFound := false
	dataChunkSize := uint32(0)

	// Continue scanning from where we left off, respecting the overall RIFF chunk size
	for bytesRead < uint64(riffChunkSize)+8 {
		n, err = io.ReadFull(r, headerBuf[:])
		if err != nil {
			if err == io.EOF && bytesRead+uint64(n) < uint64(riffChunkSize)+8 {
				// Hit EOF before finding 'data' chunk, even if RIFF suggested more data.
				break
			}
			return nil, fmt.Errorf("failed to read chunk header while searching for 'data': %w", err)
		}

		chunkID := string(headerBuf[0:4])
		chunkSize := binary.LittleEndian.Uint32(headerBuf[4:8])
		bytesRead += 8

		if chunkID == "data" {
			dataChunkSize = chunkSize
			dataChunkFound = true
			// We don't need to read the actual audio data, just its size.
			// We just need to know how much data *should* be there.
			// The reader's position is now at the start of the data payload.
			break // Found 'data' chunk
		}

		// Skip over the current chunk's data
		skipped, err := io.CopyN(io.Discard, r, int64(chunkSize))
		if err != nil {
			if err == io.EOF && skipped < int64(chunkSize) {
				// Truncated chunk data, can't determine full WAV size
				bytesRead += uint64(skipped)
				return &ScanResult{Size: bytesRead}, nil // Return what was read before truncation
			}
			return nil, fmt.Errorf("failed to skip chunk data while searching for 'data': %w", err)
		}
		bytesRead += uint64(skipped)
	}

	if !dataChunkFound {
		return nil, fmt.Errorf("missing 'data' sub-chunk")
	}

	// The total WAV size is bytesRead (which is up to the start of data payload) + dataChunkSize.
	totalWAVSize := bytesRead + uint64(dataChunkSize)

	// If the RIFF chunk size was smaller than the calculated totalWAVSize,
	// it means the file is truncated, and the actual valid data ends at the RIFF chunk boundary.
	if totalWAVSize > uint64(riffChunkSize)+8 {
		return &ScanResult{Size: uint64(riffChunkSize) + 8}, nil // Return the size declared by RIFF if data chunk extends beyond it.
	}

	// If the user wants to read the entire data chunk after this, they can do so
	// using the returned totalWAVSize or by seeking the reader if it supports it.
	// For this function, we assume the reader continues.
	// We need to advance the reader past the data chunk for the returned bytesRead to be accurate.
	skipped, err := r.Discard(int(dataChunkSize))
	if err != nil {
		if err == io.EOF && skipped < int(dataChunkSize) {
			// Data chunk is truncated. The valid WAV ends here.
			return &ScanResult{Size: bytesRead + uint64(skipped)}, nil
		}
		return nil, fmt.Errorf("failed to skip 'data' chunk: %w", err)
	}
	bytesRead += uint64(skipped)

	return &ScanResult{Size: totalWAVSize}, nil
}
