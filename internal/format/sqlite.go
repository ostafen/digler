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

const SQLiteSignature = "SQLite format 3\x00"

var sqliteFileHeader = FileHeader{
	Ext:         "sqlite",
	Description: "SQLite Database Format",
	Signatures: [][]byte{
		[]byte(SQLiteSignature),
	},
	ScanFile: ScanSQLite,
}

// ScanSqlite tries to carve a single SQLite DB starting at offset 0 in the reader.
func ScanSQLite(r *Reader) (*ScanResult, error) {
	// SQLite 3 Database Header Structure: https://www.sqlite.org/fileformat2.html#the_database_header
	// -----------------------------------------
	// Magic                (16 bytes)       "SQLite format 3\0" magic string
	// PageSize             (2 bytes)        Big-endian uint16; DB page size in bytes (must be power of 2, 512–65536)
	// FFWrite              (1 byte)         File format write version
	// FFRead               (1 byte)         File format read version
	// Reserved             (1 byte)         Reserved for future use
	// MaxEmbPayloadFrac    (1 byte)         Maximum embedded payload fraction
	// MinEmbPayloadFrac    (1 byte)         Minimum embedded payload fraction
	// LeafPayloadFrac      (1 byte)         Leaf payload fraction
	// FileChangeCounter    (4 bytes)        Big-endian uint32; file change counter
	// FileSizeInPage       (4 bytes)        Big-endian uint32; total DB size in pages
	// FirstFreelistPage    (4 bytes)        Big-endian uint32; first freelist trunk page
	// FreelistPages        (4 bytes)        Big-endian uint32; total number of freelist pages
	// SchemaCookie         (4 bytes)        Big-endian uint32; schema cookie
	// SchemaFormat         (4 bytes)        Big-endian uint32; schema format number
	// DefaultPageCacheSize (4 bytes)        Big-endian uint32; default page cache size
	// LargestRootBtree     (4 bytes)        Big-endian uint32; root b-tree page number
	// TextEncoding         (4 bytes)        Big-endian uint32; text encoding used
	// UserVersion          (4 bytes)        Big-endian uint32; user version
	// IncVacuumMode        (4 bytes)        Big-endian uint32; incremental vacuum mode flag
	// AppID                (4 bytes)        Big-endian uint32; application ID
	// ReservedForExpansion (20 bytes)       Reserved space for future expansion
	// VersionValidFor      (4 bytes)        Big-endian uint32; version valid for number
	// Version              (4 bytes)        Big-endian uint32; SQLite version number

	var hdr [100]byte
	_, err := r.Read(hdr[:])
	if err != nil {
		return nil, fmt.Errorf("failed to read SQLite header: %w", err)
	}

	if !bytes.Equal(hdr[:len(SQLiteSignature)], []byte(SQLiteSignature)) {
		return nil, fmt.Errorf("invalid SQLite magic header: expected %q, got %q", SQLiteSignature, string(hdr[:16]))
	}

	// Read PageSize: bytes 16-17 big-endian uint16
	pageSize := int(binary.BigEndian.Uint16(hdr[16:18]))
	if pageSize == 1 {
		pageSize = 65536
	}

	if !isPowerOfTwo(uint32(pageSize)) || pageSize < 512 || pageSize > 65536 {
		return nil, fmt.Errorf("invalid SQLite page size: %d", pageSize)
	}

	// Read FileChangeCounter: bytes 24-27 big-endian uint32
	fileChangeCounter := binary.BigEndian.Uint32(hdr[24:28])

	// Read FileSizeInPage: bytes 28-31 big-endian uint32
	fileSizeInPage := binary.BigEndian.Uint32(hdr[28:32])

	// Read VersionValidFor: bytes 92-95 big-endian uint32
	versionValidFor := binary.BigEndian.Uint32(hdr[92:96])

	var size uint64 = 0
	if fileSizeInPage != 0 && fileChangeCounter == versionValidFor {
		size = uint64(fileSizeInPage) * uint64(pageSize)
	}

	return &ScanResult{
		Size: size,
	}, nil
}

func isPowerOfTwo(x uint32) bool {
	return x != 0 && (x&(x-1)) == 0
}
