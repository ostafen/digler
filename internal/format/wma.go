package format

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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
// It strictly expects a valid ASF Header Object at the beginning of the reader,
// then parses its internal objects to find the definitive file size from the
// ASF File Properties Object.
func ScanWMA(r *Reader) (uint64, error) {
	var totalBytesRead uint64 // Track the total number of bytes read from the reader

	// Buffer to read GUIDs (16 bytes)
	guidBuf := make([]byte, 16)
	// Buffer to read sizes (8 bytes)
	sizeBuf := make([]byte, 8)
	// Buffer to read counts (4 bytes)
	countBuf := make([]byte, 4)

	// 1. Read and Verify ASF Header Object GUID
	if _, err := io.ReadFull(r, guidBuf); err != nil {
		return 0, fmt.Errorf("failed to read ASF header GUID: %w", err)
	}
	totalBytesRead += 16

	if !bytes.Equal(guidBuf, asfHeaderGUID) {
		return 0, errors.New("ASF header GUID not found at reader start")
	}

	// 2. Read critical fields from the ASF Header Object
	// ObjectSize (total size of this header object) is at offset 16.
	if _, err := io.ReadFull(r, sizeBuf); err != nil {
		return 0, fmt.Errorf("failed to read ASF header object size: %w", err)
	}
	totalBytesRead += 8
	headerObjectSize := binary.LittleEndian.Uint64(sizeBuf)

	// NbrHeaderObj (number of top-level objects in the header section) is at offset 24.
	if _, err := io.ReadFull(r, countBuf); err != nil {
		return 0, fmt.Errorf("failed to read ASF header object count: %w", err)
	}
	totalBytesRead += 4
	numHeaderObjects := binary.LittleEndian.Uint32(countBuf)

	// Read and discard Reserved1 and Reserved2 (1 byte each)
	if _, err := io.ReadFull(r, make([]byte, 2)); err != nil {
		return 0, fmt.Errorf("failed to read ASF header reserved bytes: %w", err)
	}
	totalBytesRead += 2

	// Basic validation of the initial header object
	if headerObjectSize < minASFHeaderObjSize || numHeaderObjects < 4 {
		return 0, errors.New("invalid ASF Header Object structure or too few internal objects")
	}

	var totalFileSize uint64  // This will store the definitive file size from File Properties Object
	var isWMAStreamFound bool // Flag to indicate if a WMA stream type is found

	// 4. Iterate through the header's sub-objects
	// `remainingHeaderBytes` tracks how many bytes are left within the main ASF Header Object.
	// We've already read `totalBytesRead` bytes of the initial header object.
	remainingHeaderBytes := headerObjectSize - totalBytesRead

	// Loop through `numHeaderObjects` or until we run out of valid bytes within the declared header.
	for i := uint32(0); i < numHeaderObjects; i++ {
		// Ensure there's enough bytes left to read the next object's basic header (GUID + ObjectSize).
		if remainingHeaderBytes < minGeneralSubObjectHeaderSize {
			break // Cannot read next object header, stop processing
		}

		// Read the sub-object's ID and Size
		if _, err := io.ReadFull(r, guidBuf); err != nil {
			return 0, fmt.Errorf("failed to read sub-object GUID: %w", err)
		}
		if _, err := io.ReadFull(r, sizeBuf); err != nil {
			return 0, fmt.Errorf("failed to read sub-object size: %w", err)
		}
		objID := guidBuf
		objSize := binary.LittleEndian.Uint64(sizeBuf)

		totalBytesRead += minGeneralSubObjectHeaderSize
		remainingHeaderBytes -= minGeneralSubObjectHeaderSize

		// Validate the sub-object's declared size
		const maxSafeObjectSize = uint64(1000 * 1024 * 1024 * 2) // 2GB as a sanity check
		if objSize < minGeneralSubObjectHeaderSize || objSize > maxSafeObjectSize {
			return 0, fmt.Errorf("invalid ASF internal object size: %d", objSize)
		}

		// Ensure the entire sub-object fits within the remaining header bytes.
		if objSize > remainingHeaderBytes {
			return 0, errors.New("malformed ASF header: sub-object extends beyond parent header boundary")
		}

		// Determine bytes to skip after processing the header of the current sub-object.
		// objSize includes the 24 bytes we just read (GUID + ObjectSize).
		bytesToSkipInObject := objSize - minGeneralSubObjectHeaderSize

		// Check for specific ASF object types
		if bytes.Equal(objID, asfFilePropGUID) {
			// Found the ASF File Properties Object
			if objSize < minFilePropObjSize { // PhotoRec's C code check for min size 0x28 (40 bytes)
				return 0, errors.New("invalid ASF File Properties Object size")
			}

			// Read up to the FileSize field.
			// We already read 24 bytes (GUID + Size). Need to skip `filePropFileSizeOffset - 24` bytes.
			skipBytes := filePropFileSizeOffset - minGeneralSubObjectHeaderSize
			if skipBytes > 0 {
				if _, err := io.ReadFull(r, make([]byte, skipBytes)); err != nil {
					return 0, fmt.Errorf("failed to skip to file size field: %w", err)
				}
				//	totalBytesRead += skipBytes
				//	bytesToSkipInObject -= skipBytes
			}

			// Read the file_size field (8 bytes).
			if _, err := io.ReadFull(r, sizeBuf); err != nil {
				return 0, fmt.Errorf("failed to read file size: %w", err)
			}
			totalBytesRead += 8
			bytesToSkipInObject -= 8

			totalFileSize = binary.LittleEndian.Uint64(sizeBuf)

			// Let's ensure the reported totalFileSize is at least big enough for the basic header.
			if totalFileSize < headerObjectSize {
				return 0, errors.New("invalid total file size in File Properties Object")
			}

		} else if bytes.Equal(objID, asfStreamPropGUID) {
			// Found the ASF Stream Properties Object
			if objSize < minStreamPropObjSize { // PhotoRec's C code check for min size 0x28 (40 bytes)
				return 0, errors.New("invalid ASF Stream Properties Object size")
			}

			// Read up to the StreamType GUID field.
			// We already read 24 bytes (GUID + Size). Need to skip `streamPropStreamTypeOffset - 24` bytes.
			skipBytes := streamPropStreamTypeOffset - minGeneralSubObjectHeaderSize
			if skipBytes > 0 {
				if _, err := io.ReadFull(r, make([]byte, skipBytes)); err != nil {
					return 0, fmt.Errorf("failed to skip to stream type field: %w", err)
				}
				//totalBytesRead += skipBytes
				//bytesToSkipInObject -= skipBytes
			}

			// Read the stream_type GUID (16 bytes).
			if _, err := io.ReadFull(r, guidBuf); err != nil {
				return 0, fmt.Errorf("failed to read stream type GUID: %w", err)
			}
			totalBytesRead += 16
			bytesToSkipInObject -= 16
			streamType := guidBuf

			if bytes.Equal(streamType, streamTypeWMA) {
				isWMAStreamFound = true
			}
		}

		// Skip any remaining bytes in the current sub-object.
		if bytesToSkipInObject > 0 {
			if _, err := io.ReadFull(r, make([]byte, bytesToSkipInObject)); err != nil {
				// This might be an io.EOF if the file is truncated, but we are within a declared object.
				// For strict parsing, this is an error, implying a malformed or incomplete object.
				return 0, fmt.Errorf("failed to skip remaining bytes in sub-object: %w", err)
			}
			totalBytesRead += bytesToSkipInObject
		}
		remainingHeaderBytes -= objSize // Decrement remaining bytes in the main header
	}

	// 5. Final Validation and Return
	// We need to have found a definitive total file size from the File Properties Object.
	if totalFileSize == 0 {
		return 0, errors.New("WMA file size not definitively determined from ASF structure")
	}

	// The declared total file size must be at least as large as the combined size of all header objects parsed.
	if totalFileSize < totalBytesRead {
		return 0, errors.New("inconsistent WMA file size: declared size is smaller than parsed header")
	}

	// For a strict WMA file, we should confirm at least one WMA audio stream exists.
	if !isWMAStreamFound {
		return 0, errors.New("no WMA audio stream found in ASF header")
	}

	return totalFileSize, nil
}
