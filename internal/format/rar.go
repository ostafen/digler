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

const RarMHDPasswordFlag = 0x0080

var (
	Rar15Signature = []byte{0x52, 0x61, 0x72, 0x21, 0x1a, 0x07, 0x00}
	Rar50Signature = []byte{0x52, 0x61, 0x72, 0x21, 0x1a, 0x07, 0x01, 0x00}
)

var rarFileHeader = FileHeader{
	Ext:         "rar",
	Description: "Rar Archive Format",
	Signatures: [][]byte{
		Rar15Signature,
		Rar50Signature,
	},
	ScanFile: ScanRAR,
}

func ScanRAR(r *Reader) (*ScanResult, error) {
	var buf [8]byte
	if _, err := r.Read(buf[:]); err != nil {
		return nil, err
	}

	if bytes.Equal(buf[:len(Rar15Signature)], Rar15Signature) {
		if err := r.UnreadByte(); err != nil {
			return nil, err
		}
		return scanRar15(r)
	}

	if bytes.Equal(buf[:len(Rar50Signature)], Rar50Signature) {
		return scanRar50(r)
	}
	return nil, fmt.Errorf("invalid RAR signature")
}

func scanRar15(r *Reader) (*ScanResult, error) {
	const Rar15ArchiveHeader byte = 0x73

	hdrType, flags, err := readRar15Block(r)
	if err != nil {
		return nil, fmt.Errorf("error reading RAR 1.5 header: %w", err)
	}

	if hdrType != Rar15ArchiveHeader {
		return nil, fmt.Errorf("invalid RAR 1.5 header type: expected 0x73, got 0x%02x", hdrType)
	}

	if (flags>>8)&RarMHDPasswordFlag != 0 {
		return nil, fmt.Errorf("RAR archive is password protected")
	}

	for {
		hdrType, _, err := readRar15Block(r)
		if err != nil {
			return nil, err
		}
		if hdrType == 0x7B {
			break
		}
	}

	return &ScanResult{
		Size: r.BytesRead(),
	}, nil
}

func readRar15Block(r *Reader) (byte, uint16, error) {
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
		return 0, 0, err
	}

	hdrType := hdrBuf[2]
	flags := binary.LittleEndian.Uint16(hdrBuf[3:5])
	if hdrType < 0x72 || hdrType > 0x7B {
		return hdrType, 0, fmt.Errorf("invalid RAR file header type: expected 0x72-0x7A or 0x7B, got 0x%02x", hdrType)
	}

	if hdrType == 0x7B { // RAR 1.5 End of Archive Header
		return hdrType, flags, nil
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
			return hdrType, flags, err
		}
	case 0x78: // Recovery Header
		var recoveryBuf [8]byte
		_, err = r.Read(recoveryBuf[:])
		if err != nil {
			return hdrType, flags, err
		}

		numBlocks := binary.LittleEndian.Uint32(recoveryBuf[:4])
		blockSize := binary.LittleEndian.Uint32(recoveryBuf[4:])
		payloadSize += numBlocks * blockSize
		n += 8
	}

	if payloadSize <= uint32(n) {
		return hdrType, flags, fmt.Errorf("invalid RAR file header: payload size %d is less than or equal to header size %d", payloadSize, n)
	}

	// Discard the rest of the header and the packed data
	_, err = r.Discard(int(payloadSize) - n)
	if err != nil {
		return hdrType, flags, fmt.Errorf("error discarding RAR file header and packed data: %w", err)
	}
	return hdrType, flags, nil
}

func scanRar50(r *Reader) (*ScanResult, error) {
	hdrType, flags, err := readRar5Block(r)
	if err != nil {
		return nil, fmt.Errorf("error reading RAR 5.0 header: %w", err)
	}
	if hdrType != 0x1 {
		return nil, fmt.Errorf("invalid RAR 5.0 header type: expected 0x1, got 0x%02x", hdrType)
	}

	if (flags>>56)&RarMHDPasswordFlag != 0 {
		return nil, fmt.Errorf("RAR 5.0 archive is password protected")
	}

	for {
		hdrType, _, err := readRar5Block(r)
		if hdrType == 0x5 {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error skipping RAR 5.0 block: %w", err)
		}
	}

	return &ScanResult{
		Size: r.BytesRead(),
	}, nil
}

func readRar5Block(r *Reader) (uint64, uint64, error) {
	// RAR 5.0 Block Structure
	// ------------------------------
	// CRC32			(4 bytes)			CRC of the block (excluding CRC field)
	// Header size		(vint)				Little-endian uint32
	// Header type		(vint)				Size of header data starting from Header type field and up to and including the optional extra area. This field must not be longer than 3 bytes in current implementation, resulting in 2 MB maximum header size.
	// Flags			(vint)				Header-specific
	// Extra area size	(optional, vint)	Present if flag 0x0001 is set
	// Data area size	(optional, vint)	Present if flag 0x0002 is set
	// Extra area	(variable bytes)		Optional data (depends on header type)
	// Data area	(variable bytes)		Optional data (file data, comments, etc.)

	_, err := r.Discard(4)
	if err != nil {
		return 0, 0, fmt.Errorf("error discarding RAR 5.0 block CRC: %w", err)
	}

	hdrSize, n, err := readRarVarInt(r)
	if err != nil {
		return 0, 0, fmt.Errorf("error reading RAR 5.0 header size: %w", err)
	}

	if n > 3 || hdrSize > 2*1024*1024 {
		return 0, 0, fmt.Errorf("invalid RAR 5.0 header size: len = %d (max 3), size = %d (max 2 MB)", n, hdrSize)
	}

	bytesRead := 0 // Since the Header Type field

	hdrType, n, err := readRarVarInt(r)
	if err != nil {
		return hdrType, 0, fmt.Errorf("error reading RAR 5.0 header type: %w", err)
	}
	bytesRead += n

	flags, n, err := readRarVarInt(r)
	if err != nil {
		return hdrType, flags, fmt.Errorf("error reading RAR 5.0 header flags: %w", err)
	}
	bytesRead += n

	totalSize := hdrSize
	if flags&0x0001 != 0 {
		// Skip the extra area size
		_, n, err := readRarVarInt(r)
		if err != nil {
			return hdrType, flags, err
		}
		bytesRead += n
	}

	if flags&0x0002 != 0 {
		// Read the data area size
		dataSize, n, err := readRarVarInt(r)
		if err != nil {
			return hdrType, flags, err
		}
		bytesRead += n
		totalSize += dataSize
	}

	discardBytes := int(totalSize) - int(bytesRead)
	if discardBytes <= 0 {
		return hdrType, flags, fmt.Errorf("invalid RAR 5.0 block size: total size %d is less than bytes read %d", totalSize, bytesRead)
	}

	_, err = r.Discard(discardBytes)
	if err != nil {
		return hdrType, flags, fmt.Errorf("error discarding RAR 5.0 block data: %w", err)
	}
	return hdrType, flags, nil
}

func readRarVarInt(r *Reader) (uint64, int, error) {
	var val uint64
	var shift uint
	var n int
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, n, err
		}
		val |= uint64(b&0x7F) << shift
		n++
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}

	// Currently RAR format uses vint to store up to 64 bit integers, resulting in 10 bytes maximum
	if n > 10 {
		return 0, -1, fmt.Errorf("invalid RAR variable-length integer: length %d exceeds maximum of 10 bytes", n)
	}
	return val, n, nil
}
