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
)

var (
	Rar15Signature = []byte{0x52, 0x61, 0x72, 0x21, 0x1a, 0x07, 0x00}
)

var rarFileHeader = FileHeader{
	Ext:         "rar",
	Description: "Rar Archive Format",
	Signatures: [][]byte{
		Rar15Signature,
	},
	ScanFile: ScanRAR,
}

func ScanRAR(r *Reader) (*ScanResult, error) {
	const (
		RarMHDPasswordFlag      = 0x0080
		Rar15HeaderSize         = 0x9
		Rar15ArchiveHeader byte = 0x73
	)

	var buf [14]byte
	if _, err := r.Read(buf[:]); err != nil {
		return nil, err
	}

	if !bytes.Equal(buf[:7], Rar15Signature) {
		return nil, fmt.Errorf("invalid RAR 1.5 signature")
	}

	// RAR 1.5 Archive Header Structure
	// ------------------------------
	// Signature (7 bytes) - "Rar!<1.5><0x07>"
	// CRC-16 of the header (2 bytes)
	// Header type (0x73) (1 byte)
	// Header flags (2 bytes)
	// Size of the header, including additional fields (2 bytes)

	if buf[0x9] != Rar15ArchiveHeader {
		return nil, fmt.Errorf("invalid RAR 1.5 header type: expected 0x73, got 0x%02x", buf[0])
	}

	if buf[0xa]&RarMHDPasswordFlag != 0 {
		return nil, fmt.Errorf("RAR archive is password protected")
	}

	hdrSize := binary.LittleEndian.Uint16(buf[0xc:0xe]) // Read the size of the header, but we don't use it here

	_, err := r.Discard(int(hdrSize) - 7) // Discard the rest of the header
	if err != nil {
		return nil, fmt.Errorf("error discarding RAR header: %w", err)
	}

	for {
		// Rar 1.5 file header structure
		// ------------------------------
		// Header CRC		(2 bytes)	CRC16 checksum of this header (excluding this field)
		// Header Type		(1 bytes)	File header type: 0x74 ('t')
		// Header Flags		(2 bytes)	Flags describing the file (encrypted, split, etc.)
		// Header Size		(2 bytes)	Size of the file header block (including variable fields)
		// Pack Size		(4 bytes)	Size of compressed data
		// Unpack Size		(4 byes)	Original uncompressed file size
		// File CRC			(4 bytes)	CRC32 checksum of the original file data
		// File Time		(4 bytes)	DOS timestamp of the file (date & time)
		// Unpack Version	(1 bytes)	Minimum software version required to unpack
		// Method			(1 bytes)	Compression method used (e.g., 0x30 = stored, 0x31+ = compressed)
		// Name Size		(2 bytes)	Length of the filename in bytes
		// File Attributes	(4 bytes)	DOS file attributes (read-only, hidden, system, archive, etc.)

		var hdrBuf [7]byte // Read header up to Pack Size
		n, err := r.Read(hdrBuf[:])
		if err != nil {
			return nil, err
		}

		hdrType := hdrBuf[2]
		flags := binary.LittleEndian.Uint16(hdrBuf[3:5])
		if hdrType < 0x72 || hdrType > 0x7B {
			return nil, fmt.Errorf("invalid RAR file header type: expected 0x72-0x7A or 0x7B, got 0x%02x", hdrType)
		}

		if hdrType == 0x7B { // RAR 1.5 End of Archive Header
			break
		}

		payloadSize := uint32(binary.LittleEndian.Uint16(hdrBuf[5:7]))
		switch hdrType {
		case 0x74, 0x75, 0x7A:
			// File Header, Comment Header and Subblock Header have a Data Size field
			err := func() error {
				if hdrType == 0x75 && flags&0x0008 == 0 {
					return nil // Skip if it's a Comment Header without the 0x0008 flag
				}
				_, err = r.Read(hdrBuf[:4]) // Read the next 4 bytes for Pack Size
				payloadSize += binary.LittleEndian.Uint32(hdrBuf[:])
				n += 4
				return err
			}()
			if err != nil {
				return nil, err
			}
		case 0x78: // Recovery Header
			var recoveryBuf [8]byte
			_, err = r.Read(recoveryBuf[:])
			if err != nil {
				return nil, err
			}

			numBlocks := binary.LittleEndian.Uint32(recoveryBuf[:4])
			blockSize := binary.LittleEndian.Uint32(recoveryBuf[4:])
			payloadSize += numBlocks * blockSize
			n += 8
		}

		if payloadSize <= uint32(n) {
			return nil, fmt.Errorf("invalid RAR file header: payload size %d is less than or equal to header size %d", payloadSize, n)
		}

		// Discard the rest of the header and the packed data
		_, err = r.Discard(int(payloadSize) - n)
		if err != nil {
			return nil, fmt.Errorf("error discarding RAR file header and packed data: %w", err)
		}
	}

	return &ScanResult{
		Size: r.BytesRead(),
	}, nil
}
