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
	"errors"
	"fmt"
	"io"
)

var pcxFileHeader = FileHeader{
	Ext:         "pcx",
	Description: "Picture Exchange Format",
	Signatures: [][]byte{
		{0x0A},
	},
	ScanFile: ScanPCX,
}

// PCXHeader represents the ZSoft PCX file header (128 bytes).
type PCXHeader struct {
	Manufacturer byte   // Manufacturer (0x0A for ZSoft .PCX)
	Version      byte   // Version (0-5, e.g., 5 for PC Paintbrush Plus)
	Encoding     byte   // Encoding (0 for uncompressed, 1 for RLE)
	BitsPerPixel byte   // Bits per pixel per plane (e.g., 1, 2, 4, 8)
	XMin         uint16 // Image dimensions (min/max X, Y)
	YMin         uint16
	XMax         uint16
	YMax         uint16
	HRes         uint16   // Horizontal resolution (DPI)
	VRes         uint16   // Vertical resolution (DPI)
	ColorMap     [48]byte // 16-color EGA palette (only if palette not separate)
	Reserved     byte     // Must be 0
	NumPlanes    byte     // Number of color planes (e.g., 1 for grayscale/indexed, 3 for RGB)
	BytesPerLine uint16   // Bytes per scanline (uncompressed, for one plane)
	PaletteType  uint16   // Palette type (1 for color/grayscale, 2 for color only)
	HScreenSize  uint16   // Horizontal screen size (used for display, often 0)
	VScreenSize  uint16   // Vertical screen size (used for display, often 0)
	Filler       [54]byte // Filler (should be 0 for older versions, variable for newer)
}

// readRLEScanline reads one RLE compressed scanline for a single plane.
// It returns the number of bytes consumed from the reader for this scanline.
func readRLEScanline(r *Reader, expectedUncompressedBytes uint16) (int, error) {
	bytesRead := 0
	decodedBytes := 0

	for decodedBytes < int(expectedUncompressedBytes) {
		b, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return bytesRead, errors.New("unexpected EOF while reading RLE data")
			}
			return bytesRead, fmt.Errorf("failed to read RLE byte: %w", err)
		}
		bytesRead++

		if b&0xC0 == 0xC0 { // If top two bits are 11 (0xC0), it's a run-length byte
			runLength := int(b & 0x3F) // Lower 6 bits are the run length
			if runLength == 0 {
				return bytesRead, errors.New("invalid RLE run length of 0")
			}
			// Read the data byte that is repeated
			_, err = r.ReadByte()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return bytesRead, errors.New("unexpected EOF while reading RLE run data byte")
				}
				return bytesRead, fmt.Errorf("failed to read RLE run data byte: %w", err)
			}

			bytesRead++
			decodedBytes += runLength
		} else { // Not a run-length byte, it's a literal data byte
			decodedBytes += 1
		}

		// The PCX specification states that a decoding break should occur at the end of each scan line.
		// This means that when a run of data is being encoded, and the end of the scan line is reached,
		// the run should stop and not continue across to the next scan line, if it is possible to stop it.
		// Since some encoders have been ignored this rule, enforcing decodedBytes to be strictly less than expectedUncompressedBytes
		// can be too restrictive for the purposes of file carving.
	}
	return bytesRead, nil
}

// ScanPCX attempts to carve a PCX file from the given io.Reader and returns its size.
// It handles both uncompressed and RLE compressed PCX files.
func ScanPCX(r *Reader) (*ScanResult, error) {
	var header PCXHeader

	// Read the entire PCX header (128 bytes)
	// We'll use a bytes.Buffer to keep track of bytes read for the header,
	// as we might need to "un-read" bytes if it's not a PCX.
	headerBuf := make([]byte, 128)
	n, err := io.ReadFull(r, headerBuf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("incomplete PCX header: end of file reached")
		}
		return nil, fmt.Errorf("failed to read PCX header: %w", err)
	}
	if n < 128 {
		return nil, fmt.Errorf("incomplete PCX header: only %d bytes read", n)
	}

	headerReader := bytes.NewReader(headerBuf)
	err = binary.Read(headerReader, binary.LittleEndian, &header)
	if err != nil {
		// This error should ideally not happen if io.ReadFull succeeded with 128 bytes
		return nil, fmt.Errorf("failed to parse PCX header: %w", err)
	}

	// Initial validations from previous version (critical for basic identification)
	if header.Manufacturer != 0x0A {
		return nil, fmt.Errorf("invalid PCX manufacturer ID: expected 0x0A, got 0x%02X", header.Manufacturer)
	}
	if header.Encoding != 0 && header.Encoding != 1 {
		return nil, fmt.Errorf("unsupported PCX encoding: expected 0 (uncompressed) or 1 (RLE), got %d", header.Encoding)
	}
	switch header.Version {
	case 0, 2, 3, 4, 5:
		// Valid versions
	default:
		return nil, fmt.Errorf("unsupported PCX version: %d", header.Version)
	}
	switch header.BitsPerPixel {
	case 1, 2, 4, 8:
		// Valid bits per pixel
	default:
		return nil, fmt.Errorf("unsupported bits per pixel: %d", header.BitsPerPixel)
	}
	if header.NumPlanes == 0 || header.NumPlanes > 4 { // Max usually 4 planes (RGB+alpha)
		return nil, fmt.Errorf("unsupported number of planes: %d", header.NumPlanes)
	}

	// Calculate image dimensions
	width := uint32(header.XMax) - uint32(header.XMin) + 1
	height := uint32(header.YMax) - uint32(header.YMin) + 1
	if width == 0 || height == 0 {
		return nil, errors.New("invalid PCX header: image dimensions (width or height) are zero")
	}
	if header.XMax < header.XMin || header.YMax < header.YMin {
		return nil, errors.New("invalid PCX header: XMax < XMin or YMax < YMin")
	}

	// Validate BytesPerLine (padded to even boundary for one plane)
	calculatedBytesPerLine := ((width*uint32(header.BitsPerPixel) + 7) / 8) // Bytes per scanline (unpadded)
	if calculatedBytesPerLine%2 != 0 {
		calculatedBytesPerLine++ // Padded to even boundary
	}
	if header.BytesPerLine < uint16(calculatedBytesPerLine) {
		return nil, fmt.Errorf("invalid PCX header: BytesPerLine (%d) is less than calculated minimum (%d)", header.BytesPerLine, calculatedBytesPerLine)
	}

	// Total bytes read so far (header size)
	totalBytesRead := uint32(128)

	// Now, handle image data based on compression
	if header.Encoding == 0 { // Uncompressed
		expectedImageDataSize := uint32(header.BytesPerLine) * uint32(header.NumPlanes) * height
		// We've only read the header. Now, skip the image data.
		skipped, err := io.CopyN(io.Discard, r, int64(expectedImageDataSize))
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, errors.New("unexpected EOF while skipping uncompressed image data")
			}
			return nil, fmt.Errorf("failed to skip uncompressed image data: %w", err)
		}
		if uint32(skipped) != expectedImageDataSize {
			return nil, fmt.Errorf("incomplete uncompressed image data: expected %d bytes, skipped %d", expectedImageDataSize, skipped)
		}
		totalBytesRead += expectedImageDataSize

	} else { // RLE compressed (Encoding == 1)
		// For RLE, we must parse the RLE data to know its actual compressed size.
		// We need to read 'height' scanlines, each composed of 'NumPlanes' planes.
		for y := uint32(0); y < height; y++ {
			for p := uint8(0); p < header.NumPlanes; p++ {
				// Each scanline for each plane is RLE encoded.
				// We need to know how many bytes to decode for this specific scanline.
				// This is `header.BytesPerLine` *for this plane*.
				consumed, err := readRLEScanline(r, header.BytesPerLine)
				if err != nil {
					return nil, fmt.Errorf("error reading RLE scanline (Y:%d, Plane:%d): %w", y, p, err)
				}
				totalBytesRead += uint32(consumed)
			}
		}
	}

	// Check for 256-color palette (version 5, 8bpp)
	if header.Version == 5 && header.BitsPerPixel == 8 {
		// These files have an optional 256-byte palette at the very end,
		// preceded by a 0x0C marker.
		marker, err := r.ReadByte()
		if err != nil {
			// If we hit EOF here, it means the file is shorter than expected,
			// or it's a valid PCX v5 8bpp without the palette marker.
			// It's common for some files to omit it, so we don't treat EOF as hard error here.
			if errors.Is(err, io.EOF) {
				return &ScanResult{Size: uint64(totalBytesRead)}, nil
			}
			return nil, fmt.Errorf("failed to read palette marker: %w", err)
		}
		totalBytesRead += 1 // Count the marker byte

		if marker == 0x0C {
			// Found the palette marker, read the 256-byte palette
			skipped, err := r.Discard(256)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil, errors.New("unexpected EOF while reading 256-byte palette")
				}
				return nil, fmt.Errorf("failed to skip 256-byte palette: %w", err)
			}
			if uint32(skipped) != 256 {
				return nil, fmt.Errorf("incomplete 256-byte palette: expected 256 bytes, skipped %d", skipped)
			}
			totalBytesRead += 256
		}
		// If the byte was not 0x0C, it means there's no 256-color palette or it's corrupted,
		// but the file might still be valid up to that point. We just count that single byte.
	}
	return &ScanResult{Size: uint64(totalBytesRead)}, nil
}
