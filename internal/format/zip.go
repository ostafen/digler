package format

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

const (
	ZipSignature4 uint32 = 0x04034B50         // ['P', 'K', 0x03, 0x04]
	ZipSignature8 uint64 = 0x30304B5004034B50 // ['P', 'K', '0', '0', 'P', 'K', 0x03, 0x04] - WinZIPv8-compressed files

	// headers
	ZipCentralDirHeader      uint32 = 0x02014B50
	ZipFileEntryHeader       uint32 = 0x04034B50
	ZipEndCentralDirHeader   uint32 = 0x06054B50
	ZipCentralDir64Header    uint32 = 0x06064B50
	ZipEndCentralDir64Header uint32 = 0x07064B50
	ZipDataDescriptorHeader  uint32 = 0x08074B50

	ZipFileEntrySize = int(unsafe.Sizeof(ZipFileEntry{}))
)

type ZipFileEntry struct {
	Version          uint16
	Flags            uint16 // This will hold all the bit-packed flags
	Compression      uint16
	LastModTime      uint16
	LastModDate      uint16
	CRC32            uint32
	CompressedSize   uint32
	UncompressedSize uint32
	FilenameLength   uint16
	ExtraLength      uint16
}

// You can define methods to access the individual bit-packed flags if needed,
// for example:

func (z *ZipFileEntry) IsEncrypted() bool {
	return (z.Flags>>0)&1 != 0 // 1st bit
}

func (z *ZipFileEntry) CompressionInfo() uint8 {
	return uint8((z.Flags >> 1) & 3) // 2nd and 3rd bits
}

func (z *ZipFileEntry) HasDescriptor() bool {
	return (z.Flags>>3)&1 != 0 // 4th bit
}

func (z *ZipFileEntry) EnhancedDeflate() bool {
	return (z.Flags>>4)&1 != 0 // 5th bit
}

func (z *ZipFileEntry) IsPatched() bool {
	return (z.Flags>>5)&1 != 0 // 6th bit
}

func (z *ZipFileEntry) StrongEncrypt() bool {
	return (z.Flags>>6)&1 != 0 // 7th bit
}

// unused2: 4 bits (bits 7-10) - no need for a dedicated getter unless you want to inspect raw unused bits

func (z *ZipFileEntry) UsesUnicode() bool {
	return (z.Flags>>11)&1 != 0 // 12th bit
}

// unused3: 1 bit (bit 12)

func (z *ZipFileEntry) EncryptedCentralDir() bool {
	return (z.Flags>>13)&1 != 0 // 14th bit
}

// unused1: 2 bits (bits 14-15)

// ScanZIP scans a byte slice for ZIP file content and estimates the total ZIP size.
func ScanZIP(buf []byte) (uint64, error) {
	r := bufio.NewReader(bytes.NewReader(buf))

	buf, err := r.Peek(4)
	if err != nil {
		return 0, fmt.Errorf("no ZIP file found")
	}

	isWinZipHeader := true
	if sig0 := binary.LittleEndian.Uint32(buf); sig0 != ZipSignature4 {
		buf, err := r.Peek(4)
		if err != nil {
			return 0, fmt.Errorf("no ZIP file found")
		}

		if sig1 := binary.LittleEndian.Uint32(buf); ((uint64(sig1) << 32) | uint64(sig0)) != ZipSignature8 {
			return 0, fmt.Errorf("invalid ZIP signature")
		}
		isWinZipHeader = false
	}

	if !isWinZipHeader {
		if err := checkZipFileEntry(r); err != nil {
			return 0, err
		}
	}

	var hdrBuf [4]byte
	for {
		_, err = r.Read(hdrBuf[:])
		if err != nil {
			return 0, nil
		}

		var err error
		switch hdr := binary.LittleEndian.Uint32(hdrBuf[:]); hdr {
		case ZipFileEntryHeader:
			err = parseZipFileEntry(r)
		case ZipCentralDirHeader:

		case ZipCentralDir64Header:

		case ZipEndCentralDirHeader:

		case ZipEndCentralDir64Header:

		}
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("not implemented yet")
	}
	return 0, nil
}

func parseZipFileEntry(r io.Reader) error {
	return nil
}

func checkZipFileEntry(r io.Reader) error {
	var e ZipFileEntry
	err := binary.Read(r, binary.LittleEndian, &e)
	if err != nil {
		return err
	}

	if e.FilenameLength == 0 || e.FilenameLength > 4096 {
		return fmt.Errorf("invalid ZIP file")
	}

	if e.Version < 10 {
		return fmt.Errorf("invalid ZIP file version")
	}
	return nil
}
