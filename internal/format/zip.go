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
	"math"
	"unsafe"
)

var zipFileHeader = FileHeader{
	Ext: "zip",
	Signatures: [][]byte{
		{'P', 'K', 0x03, 0x04},
		{'P', 'K', '0', '0', 'P', 'K', 0x03, 0x04},
	},
	ScanFile: ScanZIP,
}

var ErrInvalidZip = errors.New("invalid zip file")

const (
	// Maximum file size of a zip entry.
	MaxZipFileSize = math.MaxUint32

	// ZipSignature4 represents the standard 4-byte signature for a local file header in a ZIP file.
	ZipSignature4 uint32 = 0x04034B50 // ['P', 'K', 0x03, 0x04]
	// ZipSignature8 represents the 8-byte signature for a WinZIPv8-compressed file,
	// which includes a repeating 'PK' signature.
	ZipSignature8 uint64 = 0x30304B5004034B50 // ['P', 'K', '0', '0', 'P', 'K', 0x03, 0x04] - WinZIPv8-compressed files

	// ZIP header signatures
	// ZipCentralDirHeader is the signature for a central directory file header.
	ZipCentralDirHeader uint32 = 0x02014B50
	// ZipFileEntryHeader is the signature for a local file header.
	ZipFileEntryHeader uint32 = 0x04034B50
	// ZipEndCentralDirHeader is the signature for the end of central directory record.
	ZipEndCentralDirHeader uint32 = 0x06054B50
	// ZipCentralDir64Header is the signature for the ZIP64 central directory record.
	ZipCentralDir64Header uint32 = 0x06064B50
	// ZipEndCentralDir64Header is the signature for the ZIP64 end of central directory locator.
	ZipEndCentralDir64Header uint32 = 0x07064B50
	// ZipDataDescriptorHeader is the signature for a data descriptor, used when the CRC-32
	// and sizes are not known at the time the local file header is written.
	ZipDataDescriptorHeader uint32 = 0x08074B50

	// ZipFileEntrySize defines the fixed size of the ZipFileEntry struct in bytes.
	ZipFileEntrySize = int(unsafe.Sizeof(ZipFileEntry{}))
)

// ZipFileEntry represents the structure of a local file header in a ZIP file.
// This struct is used for reading and parsing the fixed-size portion of a file entry.
type ZipFileEntry struct {
	Version          uint16 // Version made by (and minimum version needed to extract)
	Flags            uint16 // General purpose bit flag
	Compression      uint16 // Compression method
	LastModTime      uint16 // File last modification time
	LastModDate      uint16 // File last modification date
	CRC32            uint32 // CRC-32 of uncompressed data
	CompressedSize   uint32 // Compressed size
	UncompressedSize uint32 // Uncompressed size
	FilenameLength   uint16 // Length of filename
	ExtraLength      uint16 // Length of extra field
}

// ScanZIP scans a byte slice for ZIP file content and estimates the total ZIP size.
// It reads the initial bytes to identify the ZIP signature (standard or WinZIPv8)
// and then iteratively parses file entries and central directory records to
// determine the total size of the ZIP archive.
func ScanZIP(r *Reader) (*ScanResult, error) {
	var dec zipDecoder

	if err := dec.readHeader(r); err != nil {
		return nil, err
	}

	entries := 0

	var hdrBuf [4]byte
	for {
		_, err := r.Read(hdrBuf[:])
		if err != nil {
			return nil, err
		}

		switch hdr := binary.LittleEndian.Uint32(hdrBuf[:]); hdr {
		case ZipFileEntryHeader:
			err = dec.parseZipFileEntry(r)
			entries++
		case ZipCentralDirHeader:
			if entries == 0 {
				return nil, fmt.Errorf("%w: zip file doesn't contain any file", ErrInvalidZip)
			}
			size, err := dec.parseZipCentralDir(r)
			if err != nil {
				return nil, err
			}
			return &ScanResult{
				Size: size,
				Ext:  dec.inferExt(),
			}, nil
		default:
			return nil, ErrInvalidZip
		}
		if err != nil {
			return nil, err
		}
	}
}

type zipDecoder struct {
	contentTypesSeen    bool
	relsSeen            bool
	wordDocumentSeen    bool
	pptPresentationSeen bool
	xlWorkbookSeen      bool
}

func (d *zipDecoder) readHeader(r *Reader) error {
	// Peek at the first 4 bytes to check for the standard ZIP signature.
	buf, err := r.Peek(4)
	if err != nil {
		return fmt.Errorf("%w: invalid signature", ErrInvalidZip)
	}

	if sig4 := binary.LittleEndian.Uint32(buf[:4]); sig4 != ZipSignature4 {
		// If the standard 4-byte signature is not found, peek 8 bytes to check for WinZIPv8 signature.
		_, err := r.Peek(8)
		if err != nil {
			return err
		}

		if sig8 := binary.LittleEndian.Uint64(buf); sig8 != ZipSignature8 {
			return fmt.Errorf("%w: invalid signature", ErrInvalidZip)
		}
	}
	return nil
}

// parseZipFileEntry parses a single local file entry in a ZIP file.
// It reads the fixed-size ZipFileEntry struct, followed by the filename and extra fields.
// It also handles the data descriptor if the corresponding flag is set.
func (dec *zipDecoder) parseZipFileEntry(r *Reader) error {
	var entry ZipFileEntry
	if err := binary.Read(r, binary.LittleEndian, &entry); err != nil {
		return err
	}

	var filenameBuf [math.MaxUint16]byte
	_, err := r.Read(filenameBuf[:entry.FilenameLength])
	if err != nil {
		return err
	}
	dec.processFileName(string(filenameBuf[:entry.FilenameLength]))

	// Discard the extra field bytes if ExtraLength is greater than 0.
	if entry.ExtraLength > 0 {
		_, err := r.Discard(int(entry.ExtraLength))
		if err != nil {
			return err
		}
	}

	size := entry.UncompressedSize
	if entry.Compression != 0 {
		size = entry.CompressedSize
	}

	// Check if the "has data descriptor" flag (bit 3) is set in the Flags field.
	hasDesc := entry.Flags&0x0008 != 0
	if hasDesc && size != 0 {
		// If a data descriptor is present and the size is also non-zero, it indicates an invalid state,
		// as sizes are usually zero when a data descriptor is used.
		return fmt.Errorf("invalid ZIP file entry: unexpected signature bit set")
	}

	// Handle the file data based on whether a data descriptor is present.
	if hasDesc {
		// If a data descriptor is present, seek to its signature.
		err = seekToZIPDescriptor(r)
	} else if size > 0 {
		// If no data descriptor and data size is known, discard the file data.
		_, err = r.Discard(int(size))
	}
	return err
}

// seekToZIPDescriptor searches for the data descriptor signature and discards
// the bytes until it's found. It then verifies the descriptor's signature.
func seekToZIPDescriptor(r *Reader) error {
	var zipDescriptorSignature = []byte{0x50, 0x4B, 0x07, 0x08}

	seeked, err := SeekAt(r, zipDescriptorSignature, MaxZipFileSize)
	if err != nil {
		return err
	}
	if !seeked {
		return fmt.Errorf("zip entry descriptor not found")
	}

	// Read 16 bytes (CRC-32, Compressed Size, Uncompressed Size) of the data descriptor.
	// The signature itself is 4 bytes, followed by 12 bytes of data.
	var descBuf [16]byte
	if _, err := r.Read(descBuf[:]); err != nil {
		return err
	}

	if !bytes.Equal(descBuf[:4], zipDescriptorSignature) {
		return fmt.Errorf("unable to seek at the beginning of a zip descriptor")
	}

	// NOTE: We intentionally do not validate desc.CompressedSize against the actual
	// number of bytes read or available in the stream. This is by design to allow
	// best-effort recovery during carving.

	return nil
}

// parseZipCentralDir parses the central directory record of a ZIP file.
// It searches for the end of central directory (EOCD) signature and reads
// the EOCD record to determine the total size of the ZIP archive.
func (dec *zipDecoder) parseZipCentralDir(r *Reader) (uint64, error) {
	// The signature for the end of central directory record.
	var eocdSig = []byte{0x50, 0x4B, 0x05, 0x06}

	// Seek to the EOCD signature.
	seeked, err := SeekAt(r, eocdSig, 66*1024)
	if err != nil {
		return 0, err
	}
	if !seeked {
		return 0, fmt.Errorf("unable to locate end of central directory")
	}

	// The EOCD record has a fixed size of 22 bytes after the signature,
	// which includes various fields and the comment length.
	var buf [22]byte
	_, err = r.Read(buf[:])
	if err != nil {
		return 0, err
	}

	// The comment length is the last 2 bytes of the EOCD record.
	commentLen := binary.LittleEndian.Uint16(buf[20:])
	// The total ZIP size is the number of bytes read so far by the reader,
	// plus the length of the ZIP file comment.
	return r.BytesRead() + uint64(commentLen), nil
}

func (dec *zipDecoder) processFileName(name string) {
	switch name {
	case "[Content_Types].xml":
		dec.contentTypesSeen = true
	case "_rels/.rels":
		dec.relsSeen = true
	case "word/document.xml":
		dec.wordDocumentSeen = true
	case "ppt/presentation.xml":
		dec.pptPresentationSeen = true
	case "xl/workbook.xml":
		dec.xlWorkbookSeen = true
	}
}

func (dec *zipDecoder) inferExt() string {
	isOfficeDocType := dec.contentTypesSeen && dec.relsSeen

	isWordDoc := isOfficeDocType && dec.wordDocumentSeen
	isPptDoc := isOfficeDocType && dec.pptPresentationSeen
	isXlsDoc := isOfficeDocType && dec.xlWorkbookSeen

	if isWordDoc {
		return "docx"
	}

	if isPptDoc {
		return "pptx"
	}

	if isXlsDoc {
		return "xlsx"
	}
	return "zip"
}
