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
)

const (
	tiffHeaderLittle = "\x49\x49\x2A\x00"
	tiffHeaderBig    = "\x4D\x4D\x00\x2A"
)

var tiffFileHeader = FileHeader{
	Ext:         "tif",
	Description: "Tagged Image File Format",
	Signatures: [][]byte{
		[]byte(tiffHeaderLittle),
		[]byte(tiffHeaderBig),
	},
	ScanFile: ScanTIFF,
}

func ScanTIFF(r *Reader) (*ScanResult, error) {
	const (
		tiffHeaderSize = 8
	)

	header, err := r.Peek(tiffHeaderSize)
	if err != nil {
		return nil, fmt.Errorf("not enough data for TIFF header: %w", err)
	}

	endianMarker := string(header[0:2])
	var byteOrder binary.ByteOrder
	switch endianMarker {
	case "II":
		byteOrder = binary.LittleEndian
	case "MM":
		byteOrder = binary.BigEndian
	default:
		return nil, fmt.Errorf("invalid endian marker: %x", header[0:2])
	}

	// Validate magic number
	magic := byteOrder.Uint16(header[2:4])
	if magic != 42 {
		return nil, fmt.Errorf("invalid TIFF magic number: 0x%04x", magic)
	}

	// Read offset to first IFD
	firstIFDOffset := byteOrder.Uint32(header[4:8])
	if firstIFDOffset < 8 {
		return nil, fmt.Errorf("invalid IFD offset: %d", firstIFDOffset)
	}

	// Consume the header
	_, _ = r.Discard(tiffHeaderSize)
	offset := uint64(tiffHeaderSize)

	// Seek to first IFD
	skip := int(firstIFDOffset - tiffHeaderSize)
	if skip > 0 {
		n, err := r.Discard(skip)
		if err != nil || n != skip {
			return nil, fmt.Errorf("failed to reach first IFD at offset %d", firstIFDOffset)
		}
		offset += uint64(n)
	}

	// Parse IFD chain
	for {
		var buf [4]byte

		// Read entry count (2 bytes)
		if _, err := r.Read(buf[:2]); err != nil {
			return nil, fmt.Errorf("failed to read IFD entry count: %w", err)
		}
		entryCount := byteOrder.Uint16(buf[:])
		offset += 2

		// Read all entries: each is 12 bytes
		entriesSize := int(entryCount) * 12
		if _, err := r.Discard(entriesSize); err != nil {
			return nil, fmt.Errorf("failed to skip IFD entries: %w", err)
		}
		offset += uint64(entriesSize)

		// Read next IFD offset (4 bytes)
		if _, err := r.Read(buf[:]); err != nil {
			return nil, fmt.Errorf("failed to read next IFD offset: %w", err)
		}
		nextOffset := byteOrder.Uint32(buf[:])
		offset += 4

		if nextOffset == 0 {
			break // no more IFDs
		}

		skip := int(nextOffset) - int(offset)
		if skip < 0 {
			return nil, fmt.Errorf("invalid backward IFD pointer")
		}
		if skip > 0 {
			if _, err := r.Discard(skip); err != nil {
				return nil, fmt.Errorf("failed to skip to next IFD: %w", err)
			}
			offset += uint64(skip)
		}
	}

	return &ScanResult{
		Ext:  "tif",
		Size: offset,
	}, nil
}
