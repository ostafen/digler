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

var bmpFileHeader = FileHeader{
	Ext:         "bmp",
	Description: "Bitmap Image File Format",
	Signatures: [][]byte{
		[]byte("BM"),
	},
	ScanFile: ScanBMP,
}

// BMP Compression Types
const (
	BI_RGB            = 0  // No compression
	BI_RLE8           = 1  // RLE 8-bit/pixel
	BI_RLE4           = 2  // RLE 4-bit/pixel
	BI_BITFIELDS      = 3  // RGB bit field masks (for 16bpp and 32bpp)
	BI_JPEG           = 4  // JPEG compression (Windows 95/NT and later)
	BI_PNG            = 5  // PNG compression (Windows 95/NT and later)
	BI_ALPHABITFIELDS = 6  // Alpha bit field masks (often overlaps with BI_BITFIELDS in usage)
	BI_CMYK           = 11 // CMYK uncompressed
	BI_CMYKRLE8       = 12 // CMYK RLE 8-bit/pixel
	BI_CMYKRLE4       = 13 // CMYK RLE 4-bit/pixel
)

// BMPHeader represents the BITMAPFILEHEADER structure of a BMP file.
type BMPHeader struct {
	Signature  [2]byte // BM
	FileSize   uint32  // Size of the BMP file in bytes
	Reserved1  uint16  // Must be 0
	Reserved2  uint16  // Must be 0
	DataOffset uint32  // Offset to the start of the bitmap data
}

// DIBHeader (BITMAPINFOHEADER) is the most common DIB header.
// We'll use this for initial validation.
type DIBHeader struct {
	HeaderSize      uint32 // Size of this header (should be 40 for BITMAPINFOHEADER)
	Width           int32  // Bitmap width in pixels
	Height          int32  // Bitmap height in pixels
	Planes          uint16 // Number of color planes (must be 1)
	BitsPerPixel    uint16 // Number of bits per pixel (e.g., 1, 4, 8, 16, 24, 32)
	Compression     uint32 // Compression method
	ImageSize       uint32 // Size of the raw bitmap data (can be 0 for BI_RGB)
	XPixelsPerMeter int32
	YPixelsPerMeter int32
	ColorsUsed      uint32 // Number of colors in the color palette
	ColorsImportant uint32 // Number of important colors
}

// ScanBMP attempts to carve a BMP from the given Reader and returns its size.
// It performs more extensive checks to reduce false positives.
func ScanBMP(r *Reader) (*ScanResult, error) {
	var bmpHeader BMPHeader
	var dibHeader DIBHeader

	// Read the BMP File Header (14 bytes)
	err := binary.Read(r, binary.LittleEndian, &bmpHeader)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("incomplete BMP header: end of file reached")
		}
		return nil, fmt.Errorf("failed to read BMP file header: %w", err)
	}

	// Basic validation of BMP File Header
	if bmpHeader.Signature[0] != 'B' || bmpHeader.Signature[1] != 'M' {
		return nil, errors.New("invalid BMP signature: expected 'BM'")
	}
	if bmpHeader.Reserved1 != 0 || bmpHeader.Reserved2 != 0 {
		return nil, errors.New("invalid BMP header: reserved fields are not zero")
	}
	if bmpHeader.FileSize < 14+40 { // Minimum size: BMP header (14) + BITMAPINFOHEADER (40)
		return nil, errors.New("invalid BMP header: file size too small to contain basic headers")
	}
	if bmpHeader.DataOffset < 14 { // Data offset must be at least after BMP header
		return nil, errors.New("invalid BMP header: data offset is before the BMP file header")
	}

	// Read the DIB Header (assume BITMAPINFOHEADER for commonality, 40 bytes)
	// We need to peek at the HeaderSize field to determine the exact DIB header type.
	// For simplicity, we'll assume BITMAPINFOHEADER (40 bytes) and then read it.
	// A more robust solution might read only the HeaderSize first, then seek/read the rest.
	var buf [124]byte // max size of header (BITMAPV5HEADER)
	n, err := r.Read(buf[:4])
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("incomplete DIB header: end of file reached")
		}
		return nil, fmt.Errorf("failed to read DIB header size: %w", err)
	}
	if n < 4 {
		return nil, errors.New("incomplete DIB header: not enough bytes for header size")
	}
	dibHeader.HeaderSize = binary.LittleEndian.Uint32(buf[:])

	// Validate DIB Header fields
	if dibHeader.HeaderSize != 40 &&
		dibHeader.HeaderSize != 12 && // BITMAPCOREHEADER
		dibHeader.HeaderSize != 64 && // BITMAPINFOHEADER V2
		dibHeader.HeaderSize != 108 && // BITMAPV4HEADER
		dibHeader.HeaderSize != 124 { // BITMAPV5HEADER
		return nil, fmt.Errorf("unsupported DIB header size: %d", dibHeader.HeaderSize)
	}

	// Read the rest of the DIBHeader based on its size
	n, err = r.Read(buf[4:dibHeader.HeaderSize])
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("incomplete DIB header: end of file reached while reading full header")
		}
		return nil, fmt.Errorf("failed to read remaining DIB header: %w", err)
	}
	if n < int(dibHeader.HeaderSize-4) {
		return nil, errors.New("incomplete DIB header: not enough bytes for full header")
	}

	// Create a new reader for the remaining DIB header bytes
	dibReader := bytes.NewReader(buf[:dibHeader.HeaderSize])
	err = binary.Read(dibReader, binary.LittleEndian, &dibHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DIB header: %w", err)
	}

	if dibHeader.Planes != 1 {
		return nil, errors.New("invalid DIB header: number of planes must be 1")
	}

	// Validate BitsPerPixel (common values)
	switch dibHeader.BitsPerPixel {
	case 1, 4, 8, 16, 24, 32:
		// Valid
	default:
		return nil, fmt.Errorf("unsupported bits per pixel: %d", dibHeader.BitsPerPixel)
	}

	// Check if the compression type is a recognized BMP compression method.
	// If it's recognized, we proceed to handle it (or state if not yet fully supported).
	// If it's an unknown value, it's a strong indicator of a malformed file.
	switch dibHeader.Compression {
	case BI_RGB, BI_RLE8, BI_RLE4, BI_BITFIELDS, BI_JPEG, BI_PNG,
		BI_ALPHABITFIELDS, BI_CMYK, BI_CMYKRLE8, BI_CMYKRLE4:
	default:
		return nil, fmt.Errorf("unrecognized or unsupported BMP compression type: %d", dibHeader.Compression)
	}

	// Validate image dimensions (should be positive)
	if dibHeader.Width <= 0 || dibHeader.Height == 0 { // Height can be negative for top-down DIBs, but 0 is invalid
		return nil, errors.New("invalid DIB header: image dimensions are invalid")
	}

	// Consistency check: DataOffset vs. HeaderSize + BmpHeaderSize
	// For simple uncompressed BMPs, DataOffset should be (BMPHeaderSize + DIBHeaderSize + PaletteSize)
	// PaletteSize depends on BitsPerPixel and ColorsUsed.
	expectedMinDataOffset := uint32(14) + dibHeader.HeaderSize
	if dibHeader.BitsPerPixel <= 8 && dibHeader.ColorsUsed == 0 {
		// If palette exists and ColorsUsed is 0, it means 2^BitsPerPixel colors.
		expectedMinDataOffset += (1 << dibHeader.BitsPerPixel) * 4 // 4 bytes per palette entry (BGR0)
	} else if dibHeader.BitsPerPixel <= 8 && dibHeader.ColorsUsed > 0 {
		expectedMinDataOffset += dibHeader.ColorsUsed * 4
	}

	if bmpHeader.DataOffset < expectedMinDataOffset {
		return nil, fmt.Errorf("invalid BMP header: data offset (%d) is less than expected minimum (%d)", bmpHeader.DataOffset, expectedMinDataOffset)
	}

	// One more check: calculate expected image data size and compare with FileSize.
	// This is approximate for uncompressed images and can vary with compression.
	// Also, each scanline is padded to a 4-byte boundary.
	bytesPerPixel := uint32(dibHeader.BitsPerPixel / 8)
	if dibHeader.BitsPerPixel%8 != 0 { // For 1, 4 bit per pixel
		bytesPerPixel = 1 // Simplified: at least 1 byte per pixel for partial bytes
		if dibHeader.BitsPerPixel == 1 {
			bytesPerPixel = 1 // 8 pixels per byte
		} else if dibHeader.BitsPerPixel == 4 {
			bytesPerPixel = 1 // 2 pixels per byte
		}
	}

	rowSize := uint32(dibHeader.Width) * bytesPerPixel
	paddedRowSize := (rowSize + 3) & ^uint32(3) // Pad to 4-byte boundary

	// For 1, 4, 8 bpp, rowSize is in pixels / (8/bpp).
	// Example: 1bpp, 8 pixels per byte. Width 10. rowSize = ceil(10/8) = 2 bytes. paddedRowSize = 4 bytes.
	if dibHeader.BitsPerPixel == 1 {
		rowSize = (uint32(dibHeader.Width) + 7) / 8 // Bytes per row (unpadded)
	} else if dibHeader.BitsPerPixel == 4 {
		rowSize = (uint32(dibHeader.Width) + 1) / 2 // Bytes per row (unpadded)
	}

	// Recalculate paddedRowSize for these cases
	if dibHeader.BitsPerPixel <= 8 {
		paddedRowSize = (rowSize + 3) &^ 3
	}

	// Consider the image height. Height can be negative for top-down images.
	absHeight := uint32(dibHeader.Height)
	if dibHeader.Height < 0 {
		absHeight = uint32(-dibHeader.Height)
	}

	expectedImageDataSize := paddedRowSize * absHeight

	// This check is tricky because ImageSize can be 0 for BI_RGB (uncompressed)
	// and is only mandatory for compressed BMPs. If ImageSize is provided, we can use it.
	if dibHeader.ImageSize != 0 && dibHeader.ImageSize < expectedImageDataSize {
		// If ImageSize is provided, it must be at least the calculated uncompressed size.
		// However, it could be larger due to padding or other factors for compressed data.
		// This check is a strong indicator of corruption if ImageSize is too small.
		return nil, fmt.Errorf("invalid DIB header: image size (%d) is less than calculated minimum (%d)", dibHeader.ImageSize, expectedImageDataSize)
	}

	// FileSize = BMPHeaderSize + DIBHeaderSize + PaletteSize + ImageDataSize
	// For BI_RGB and ImageSize == 0, use calculated expectedImageDataSize.
	// For BI_RGB and ImageSize != 0, use ImageSize.
	// For compressed BMPs, the ImageSize is usually the actual compressed size.
	var actualImageDataSize uint32
	if dibHeader.Compression == BI_RGB {
		actualImageDataSize = expectedImageDataSize
	} else {
		actualImageDataSize = dibHeader.ImageSize
	}

	// The offset to bitmap data is from the start of the file.
	// The total file size should be at least (DataOffset + Image Data Size).
	// It's possible for the actual file size to be larger due to application-specific
	// trailing data, but it should not be smaller than the stated content.
	expectedTotalSize := bmpHeader.DataOffset + actualImageDataSize
	if bmpHeader.FileSize < expectedTotalSize {
		return nil, fmt.Errorf("inconsistent file size: header states %d, but expected at least %d based on data offset and image size", bmpHeader.FileSize, expectedTotalSize)
	}
	return &ScanResult{Size: uint64(bmpHeader.FileSize)}, nil
}
