package format

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

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
	// This is based on typical structure and checks in PhotoRec's C code (0x28 or 40 bytes).
	minFilePropObjSize = 40

	// filePropFileSizeOffset is the byte offset of the FileSize field within the ASF File Properties Object,
	// relative to the start of the File Properties Object's data (after its own 24-byte header).
	// Calculated as: 16 (FileID) = 16.
	// So, from the *start* of the File Properties Object: 16 (GUID) + 8 (ObjectSize) + 16 (FileID) = 40.
	filePropFileSizeOffset = 40

	// minStreamPropObjSize is the minimum expected size for an ASF Stream Properties Object.
	// Based on PhotoRec's C code (0x28 or 40 bytes).
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
func ScanWMA(buf []byte) (uint64, error) {
	bufLen := uint64(len(buf)) // Use uint64 for consistent comparisons

	// 1. Initial Buffer Size Check
	if bufLen < minASFHeaderObjSize {
		return 0, errors.New("buffer too small for ASF header")
	}

	// 2. Verify ASF Header Object GUID
	if !bytes.Equal(buf[:16], asfHeaderGUID) {
		return 0, errors.New("ASF header GUID not found at buffer start")
	}

	// 3. Read critical fields from the ASF Header Object
	// ObjectSize (total size of this header object) is at offset 16.
	headerObjectSize := binary.LittleEndian.Uint64(buf[16:24])
	// NbrHeaderObj (number of top-level objects in the header section) is at offset 24.
	numHeaderObjects := binary.LittleEndian.Uint32(buf[24:28])

	// Basic validation of the initial header object
	// The C code checked `headerObjectSize < 30` and `numHeaderObjects < 4`.
	if headerObjectSize < minASFHeaderObjSize || numHeaderObjects < 4 {
		return 0, errors.New("invalid ASF Header Object structure or too few internal objects")
	}

	// If the declared header object size exceeds the buffer, it's truncated at the header level.
	if headerObjectSize > bufLen {
		return bufLen, nil // Return available data as per strict carving rules
	}

	var totalFileSize uint64  // This will store the definitive file size from File Properties Object
	var isWMAStreamFound bool // Flag to indicate if a WMA stream type is found

	// 4. Iterate through the header's sub-objects
	// `currentOffset` tracks the start of the current sub-object being parsed.
	currentOffset := uint64(minASFHeaderObjSize) // Start after the initial ASF Header Object's fixed part

	// Loop through `numHeaderObjects` or until we run out of valid buffer within the declared header.
	for i := uint32(0); i < numHeaderObjects; i++ {
		// Ensure there's enough buffer to read the next object's basic header (GUID + ObjectSize).
		if currentOffset+minGeneralSubObjectHeaderSize > bufLen {
			break // Cannot read next object header, stop processing
		}
		// Also ensure the current object doesn't spill past the main Header Object's declared boundary.
		if currentOffset+minGeneralSubObjectHeaderSize > headerObjectSize {
			return 0, errors.New("malformed ASF header: sub-object extends beyond parent header")
		}

		// Read the sub-object's ID and Size
		objID := buf[currentOffset : currentOffset+16]
		objSize := binary.LittleEndian.Uint64(buf[currentOffset+16 : currentOffset+24])

		// Validate the sub-object's declared size
		// C code used `objSize < 24 || objSize > 0x8000000000000000`. We'll use a pragmatic max.
		const maxSafeObjectSize = uint64(1000 * 1024 * 1024 * 2) // 2GB as a sanity check
		if objSize < minGeneralSubObjectHeaderSize || objSize > maxSafeObjectSize {
			return 0, fmt.Errorf("invalid ASF internal object size: %d", objSize)
		}

		// Ensure the entire sub-object fits within the buffer. If not, the file is truncated.
		if currentOffset+objSize > bufLen {
			return bufLen, nil // Return available length as per strict carving rules
		}
		// Also ensure it fits within the main Header Object's declared bounds.
		if currentOffset+objSize > headerObjectSize {
			return 0, errors.New("malformed ASF header: sub-object extends beyond header boundary")
		}

		// Check for specific ASF object types
		if bytes.Equal(objID, asfFilePropGUID) {
			// Found the ASF File Properties Object
			if objSize < minFilePropObjSize { // PhotoRec's C code check for min size 0x28 (40 bytes)
				return 0, errors.New("invalid ASF File Properties Object size")
			}

			// The file_size field is at offset `filePropFileSizeOffset` within this object.
			fileSizeFieldOffset := currentOffset + filePropFileSizeOffset
			if fileSizeFieldOffset+8 > bufLen || fileSizeFieldOffset+8 > currentOffset+objSize {
				return 0, errors.New("truncated ASF File Properties Object for 'file_size'")
			}
			totalFileSize = binary.LittleEndian.Uint64(buf[fileSizeFieldOffset : fileSizeFieldOffset+8])

			// PhotoRec C code checked `size < 30+104` (30 is minASFHeaderObjSize). This is a heuristic.
			// Let's ensure the reported totalFileSize is at least big enough for the basic header.
			if totalFileSize < headerObjectSize { // At least the size of the initial header block
				return 0, errors.New("invalid total file size in File Properties Object")
			}
		} else if bytes.Equal(objID, asfStreamPropGUID) {
			// Found the ASF Stream Properties Object
			if objSize < minStreamPropObjSize { // PhotoRec's C code check for min size 0x28 (40 bytes)
				return 0, errors.New("invalid ASF Stream Properties Object size")
			}

			// Check if this is a WMA stream type
			streamTypeFieldOffset := currentOffset + streamPropStreamTypeOffset
			if streamTypeFieldOffset+16 > bufLen || streamTypeFieldOffset+16 > currentOffset+objSize {
				return 0, errors.New("truncated ASF Stream Properties Object for 'stream_type'")
			}
			streamType := buf[streamTypeFieldOffset : streamTypeFieldOffset+16]

			if bytes.Equal(streamType, streamTypeWMA) {
				isWMAStreamFound = true
			}
			// We could check for WMV stream types here too if needed for broader ASF detection.
		}

		currentOffset += objSize // Move to the start of the next sub-object
	}

	// 5. Final Validation and Return
	// We need to have found a definitive total file size from the File Properties Object.
	if totalFileSize == 0 {
		return 0, errors.New("WMA file size not definitively determined from ASF structure")
	}

	// PhotoRec's C code also had a check: `if(size > 0 && size < offset_prop) return 0;`
	// This means the declared total file size must be at least as large as the combined size of all header objects parsed.
	if totalFileSize < currentOffset {
		return 0, errors.New("inconsistent WMA file size: declared size is smaller than parsed header")
	}

	// For a strict WMA file, we should confirm at least one WMA audio stream exists.
	if !isWMAStreamFound {
		return 0, errors.New("no WMA audio stream found in ASF header")
	}

	// If the calculated total file size extends beyond the buffer, it's a truncated file.
	if totalFileSize > bufLen {
		return bufLen, nil // Return the end of the available data
	}
	return totalFileSize, nil
}
