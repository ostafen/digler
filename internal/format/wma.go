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
)

var wmaFileHeader = FileHeader{
	Ext:         "wma",
	Description: "Windows Media Audio Format",
	Signatures: [][]byte{
		asfHeaderGUID,
	},
	ScanFile: ScanWMA,
}

// GUIDs for ASF objects (WMA/WMV are built on ASF)
// All values are Little Endian as they appear in the file.
var (
	// asfHeaderGUID: {75B22630-668E-11CF-A6D9-00AA0062CE6C}
	// This is the mandatory first object in an ASF file.
	asfHeaderGUID = []byte{
		0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11,
		0xA6, 0xD9, 0x00, 0xAA, 0x00, 0x62, 0xCE, 0x6C,
	}

	// asfFilePropGUID: {8CABDCA1-A947-11CF-8EE4-00C00C205365}
	// Contains global file attributes, including total file size.
	asfFilePropGUID = []byte{
		0xA1, 0xDC, 0xAB, 0x8C, 0x47, 0xA9, 0xCF, 0x11,
		0x8E, 0xE4, 0x00, 0xC0, 0x0C, 0x20, 0x53, 0x65,
	}

	// asfStreamPropGUID: {B7DC0791-A9B7-11CF-8EE6-00C00C205365}
	// Defines properties for a single stream within the ASF file.
	asfStreamPropGUID = []byte{
		0x91, 0x07, 0xDC, 0xB7, 0xB7, 0xA9, 0xCF, 0x11,
		0x8E, 0xE6, 0x00, 0xC0, 0x0C, 0x20, 0x53, 0x65,
	}

	// streamTypeWMA: {F8699E40-5B4D-11CF-A8FD-00805F5C442B}
	// Identifies a Windows Media Audio stream.
	streamTypeWMA = []byte{
		0x40, 0x9E, 0x69, 0xF8, 0x4D, 0x5B, 0xCF, 0x11,
		0xA8, 0xFD, 0x00, 0x80, 0x5F, 0x5C, 0x44, 0x2B,
	}
)

const (
	// minASFHeaderObjSize is the minimum byte size for the initial ASF Header Object itself.
	// 16 (GUID) + 8 (ObjectSize) + 4 (NumHeaderObjects) + 1 (Reserved1) + 1 (Reserved2) = 30 bytes.
	minASFHeaderObjSize = 30

	// minGeneralSubObjectHeaderSize is the minimum size to read any ASF object's header (GUID + ObjectSize).
	minGeneralSubObjectHeaderSize = 24

	// minFilePropObjSize is the minimum expected size for an ASF File Properties Object.
	minFilePropObjSize = 40

	// filePropFileSizeOffset is the byte offset of the FileSize field within the ASF File Properties Object,
	// relative to the start of the File Properties Object's data (after its own 24-byte header).
	// Calculated as: 16 (FileID) = 16.
	// So, from the *start* of the File Properties Object: 16 (GUID) + 8 (ObjectSize) + 16 (FileID) = 40.
	filePropFileSizeOffset = 40

	// minStreamPropObjSize is the minimum expected size for an ASF Stream Properties Object.
	minStreamPropObjSize = 40

	// streamPropStreamTypeOffset is the byte offset of the StreamType GUID field within the ASF Stream Properties Object,
	// relative to the start of the Stream Properties Object's data (after its own 24-byte header).
	// Calculated as: 16 (StreamType GUID) = 16.
	// So, from the *start* of the Stream Properties Object: 16 (GUID) + 8 (ObjectSize) = 24. This is the start of the data.
	// Then +0 for StreamType. So it's at offset 24 from the start of the object.
	streamPropStreamTypeOffset = 24
)

// ScanWMA attempts to validate a WMA file in ASF format and return its total size.
// It strictly expects a valid ASF Header Object at the beginning of the buffer,
// then parses its internal objects to find the definitive file size from the
// ASF File Properties Object.
func ScanWMA(r *Reader) (*ScanResult, error) {
	var buf [minASFHeaderObjSize]byte
	_, err := r.Read(buf[:])
	if err != nil {
		return nil, nil
	}

	// Verify ASF Header Object GUID
	if !bytes.Equal(buf[:16], asfHeaderGUID) {
		return nil, errors.New("ASF header GUID not found at buffer start")
	}

	// Read critical fields from the ASF Header Object
	// ObjectSize (total size of this header object) is at offset 16.
	headerObjectSize := binary.LittleEndian.Uint64(buf[16:24])
	// NbrHeaderObj (number of top-level objects in the header section) is at offset 24.
	numHeaderObjects := binary.LittleEndian.Uint32(buf[24:28])

	// Basic validation of the initial header object
	// The C code checked `headerObjectSize < 30` and `numHeaderObjects < 4`.
	if headerObjectSize < minASFHeaderObjSize || numHeaderObjects < 4 {
		return nil, errors.New("invalid ASF Header Object structure or too few internal objects")
	}

	var totalFileSize uint64  // This will store the definitive file size from File Properties Object
	var isWMAStreamFound bool // Flag to indicate if a WMA stream type is found

	// Iterate through the header's sub-objects
	// `currentOffset` tracks the start of the current sub-object being parsed.

	bytesRead := uint64(minASFHeaderObjSize) // Start after the initial ASF Header Object's fixed part

	// Loop through `numHeaderObjects` or until we run out of valid buffer within the declared header.
	for i := uint32(0); i < numHeaderObjects; i++ {
		// Also ensure the current object doesn't spill past the main Header Object's declared boundary.
		if bytesRead+minGeneralSubObjectHeaderSize > headerObjectSize {
			return nil, errors.New("malformed ASF header: sub-object extends beyond parent header")
		}

		_, err := r.Read(buf[:minGeneralSubObjectHeaderSize])
		if err != nil {
			return nil, err
		}

		// Read the sub-object's ID and Size
		objID := buf[:16]
		objSize := binary.LittleEndian.Uint64(buf[16:24])

		// Validate the sub-object's declared size
		// C code used `objSize < 24 || objSize > 0x8000000000000000`. We'll use a pragmatic max.

		const maxSafeObjectSize = uint64(1000 * 1024 * 1024 * 2) // 2GB as a sanity check
		if objSize < minGeneralSubObjectHeaderSize || objSize > maxSafeObjectSize {
			return nil, fmt.Errorf("invalid ASF internal object size: %d", objSize)
		}

		// Also ensure it fits within the main Header Object's declared bounds.
		if bytesRead+objSize > headerObjectSize {
			return nil, errors.New("malformed ASF header: sub-object extends beyond header boundary")
		}

		// Check for specific ASF object types
		if bytes.Equal(objID, asfFilePropGUID) {
			// Found the ASF File Properties Object
			if objSize < minFilePropObjSize {
				return nil, errors.New("invalid ASF File Properties Object size")
			}

			buf, err := r.Peek(minFilePropObjSize - minGeneralSubObjectHeaderSize)
			if err != nil {
				return nil, err
			}

			// The file_size field is at offset `filePropFileSizeOffset` within this object.
			fileSizeFieldOffset := bytesRead + filePropFileSizeOffset
			//fileSizeFieldOffset+8 > bufLen ||
			if fileSizeFieldOffset+8 > bytesRead+objSize {
				return nil, errors.New("truncated ASF File Properties Object for 'file_size'")
			}

			totalFileSize = binary.LittleEndian.Uint64(buf[16 : 16+8])

			// Let's ensure the reported totalFileSize is at least big enough for the basic header.
			if totalFileSize < headerObjectSize { // At least the size of the initial header block
				return nil, errors.New("invalid total file size in File Properties Object")
			}
		} else if bytes.Equal(objID, asfStreamPropGUID) {
			buf, err := r.Peek(minFilePropObjSize - minStreamPropObjSize)
			if err != nil {
				return nil, err
			}
			// Found the ASF Stream Properties Object
			if objSize < minStreamPropObjSize {
				return nil, errors.New("invalid ASF Stream Properties Object size")
			}

			// Check if this is a WMA stream type
			streamTypeFieldOffset := bytesRead + streamPropStreamTypeOffset

			//streamTypeFieldOffset+16 > bufLen
			if streamTypeFieldOffset+16 > bytesRead+objSize {
				return nil, errors.New("truncated ASF Stream Properties Object for 'stream_type'")
			}

			streamType := buf[:16]
			if bytes.Equal(streamType, streamTypeWMA) {
				isWMAStreamFound = true
			}
			// We could check for WMV stream types here too if needed for broader ASF detection.
		}

		_, err = r.Discard(int(objSize - 24))
		if err != nil {
			return nil, err
		}
		bytesRead += objSize // Move to the start of the next sub-object
	}

	// We need to have found a definitive total file size from the File Properties Object.
	if totalFileSize == 0 {
		return nil, errors.New("WMA file size not definitively determined from ASF structure")
	}

	// Declared total file size must be at least as large as the combined size of all header objects parsed.
	if totalFileSize < bytesRead {
		return nil, errors.New("inconsistent WMA file size: declared size is smaller than parsed header")
	}

	// For a strict WMA file, we should confirm at least one WMA audio stream exists.
	if !isWMAStreamFound {
		return nil, errors.New("no WMA audio stream found in ASF header")
	}

	// If the calculated total file size extends beyond the buffer, it's a truncated file.
	return &ScanResult{Size: totalFileSize}, nil

}
