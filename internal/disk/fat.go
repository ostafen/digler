package disk // Or choose a more specific package name like 'fat' or 'filesystem/fat'

import (
	"bytes"
	"encoding/binary" // For reading multi-byte values (like sector_size, fat_length)
	"fmt"
	// For Sizeof to check struct sizes (optional, for debugging)
)

// --- Constants ---

// Partition type indicators
const (
	FAT1X_PART_NAME = 0x2B // Often for FAT12/16 partitions
	FAT32_PART_NAME = 0x47 // For FAT32 partitions
)

// FAT filesystem specific names/markers
const (
	FAT_NAME1 = 0x36 // Likely a marker related to FAT12/16 detection
	FAT_NAME2 = 0x52 // FAT32 only marker
)

// File/Directory Entry Flags
const (
	DELETED_FLAG = 0xE5 // Marks a file/directory as deleted when in name[0]
)

// File/Directory Attributes (bit flags)
const (
	ATTR_RO     = 1  // Read-only
	ATTR_HIDDEN = 2  // Hidden
	ATTR_SYS    = 4  // System file
	ATTR_VOLUME = 8  // Volume label entry
	ATTR_DIR    = 16 // Directory
	ATTR_ARCH   = 32 // Archive bit
)

// Combined Attribute Masks
const (
	ATTR_NONE     = 0                                                                     // No attributes
	ATTR_UNUSED   = ATTR_VOLUME | ATTR_ARCH | ATTR_SYS | ATTR_HIDDEN                      // Attributes that are copied "as is" or potentially unused
	ATTR_EXT      = ATTR_RO | ATTR_HIDDEN | ATTR_SYS | ATTR_VOLUME                        // Extended attributes (Windows 95/NT)
	ATTR_EXT_MASK = ATTR_RO | ATTR_HIDDEN | ATTR_SYS | ATTR_VOLUME | ATTR_DIR | ATTR_ARCH // Mask for all extended attributes
)

// FAT Chain End-Of-Cluster (EOC) and Bad Cluster Markers
const (
	FAT12_BAD = 0x0FF7
	FAT12_EOC = 0x0FF8
	FAT16_BAD = 0xFFF7
	FAT16_EOC = 0xFFF8
	FAT32_BAD = 0x0FFFFFF7
	FAT32_EOC = 0x0FFFFFF8
)

// Common Boot Sector Size
const Fat1xBootSectorSize = 0x200 // 512 bytes

// FatBootSector represents the FAT partition boot sector (BIOS Parameter Block - BPB).
// This struct maps to the C `struct fat_boot_sector`.
// Fields that are `uint16_t` or `uint32_t` in C are represented as byte arrays in Go,
// to explicitly handle endianness when reading from a raw byte slice.
type FatBootSector struct {
	Ignored           [3]byte // 0x00 Boot strap short or near jump
	SystemID          [8]int8 // 0x03 Name - can be used to special case partition manager volumes
	SectorSize        uint16  // 0x0B Bytes per logical sector (uint16_t)
	SectorsPerCluster uint8   // 0x0D Sectors/cluster
	Reserved          uint16  // 0x0E Reserved sectors (uint16_t)
	Fats              uint8   // 0x10 Number of FATs
	DirEntries        uint16  // 0x11 Root directory entries (uint16_t)
	Sectors           uint16  // 0x13 Number of sectors (uint16_t)
	Media             uint8   // 0x15 Media code (unused)
	FatLength         uint16  // 0x16 Sectors/FAT (uint16_t)
	SecsTrack         uint16  // 0x18 Sectors per track (uint16_t)
	Heads             uint16  // 0x1A Number of heads (uint16_t)
	Hidden            uint32  // 0x1C Hidden sectors (unused, uint32_t)
	TotalSect         uint32  // 0x20 Total number of sectors (if sectors == 0, uint32_t)

	// The following fields are only used by FAT32
	Fat32Length  uint32   // 0x24 Sectors/FAT (uint32_t)
	Flags        uint16   // 0x28 Bit 8: FAT mirroring, low 4: active FAT (uint16_t)
	Version      uint16   // 0x2A Major, minor filesystem version (uint8_t[2])
	RootCluster  [4]byte  // 0x2C First cluster in root directory (uint32_t)
	InfoSector   uint16   // 0x30 Filesystem info sector (uint16_t)
	BackupBoot   uint16   // 0x32 Backup boot sector (uint16_t)
	BPBReserved  [12]byte // 0x34 Unused (uint8_t[12])
	BSDrvNum     uint8    // 0x40 Drive number
	BSReserved1  uint8    // 0x41 Reserved
	BSBootSig    uint8    // 0x42 Extended boot signature (0x29)
	BSVolID      [4]byte  // 0x43 Volume serial number (uint32_t)
	BSVolLab     [11]byte // 0x47 Volume label
	BSFilSysType [8]byte  // 0x52 Filesystem type ("FAT12   ", "FAT16   ", "FAT32   ")

	// Rest of the boot sector padding and marker
	Nothing [420]byte // 0x5A Padding
	Marker  uint16    // 0x1FE Boot sector signature (0xAA55, uint16_t)
}

// ReadRootCluster returns the first cluster of the root directory (for FAT32).
func (b *FatBootSector) ReadRootCluster() uint32 {
	return binary.LittleEndian.Uint32(b.RootCluster[:])
}

/*
// String provides a human-readable representation of the FatBootSector (for debugging).
func (b *FatBootSector) String() string {
	return fmt.Sprintf("FAT Boot Sector:\n"+
		"  System ID: %s\n"+
		"  Sector Size: %d bytes\n"+
		"  Sectors Per Cluster: %d\n"+
		"  Reserved Sectors: %d\n"+
		"  Number of FATs: %d\n"+
		"  Root Dir Entries: %d\n"+
		"  Total Sectors (16-bit): %d\n"+
		"  Media Type: 0x%02X\n"+
		"  FAT Length (16-bit): %d\n"+
		"  Total Sectors (32-bit): %d\n"+
		"  FAT32 Length: %d\n"+
		"  FAT32 Root Cluster: %d\n"+
		"  FS Info Sector: %d\n"+
		"  Backup Boot Sector: %d\n"+
		"  Volume Label: %s\n"+
		"  Filesystem Type: %s\n"+
		"  Marker: 0x%04X",
		b.SystemID, b.ReadSectorSize(), b.SectorsPerCluster, b.ReadReservedSectors(),
		b.Fats, b.ReadDirEntries(), b.ReadSectors(), b.Media, b.ReadFatLength(),
		b.ReadTotalSect(), b.ReadFat32Length(), b.ReadRootCluster(),
		b.ReadInfoSector(), b.ReadBackupBoot(),
		b.BSVolLab, b.BSFilSysType, b.ReadMarker())
}
*/

func ReadFatBootSectorFrom(data []byte) (*FatBootSector, error) {
	if len(data) != Fat1xBootSectorSize {
		return nil, fmt.Errorf("input data slice size mismatch: expected %d bytes, got %d bytes",
			Fat1xBootSectorSize, len(data))
	}

	var bs FatBootSector
	r := bytes.NewReader(data)

	err := binary.Read(r, binary.LittleEndian, &bs)
	if err != nil {
		return nil, fmt.Errorf("error reading into FatBootSector with binary.Read: %w", err)
	}

	if bs.Marker != 0xAA55 {
		return nil, fmt.Errorf("invalid boot sector marker: expected 0xAA55, got 0x%04X", bs.Marker)
	}
	return &bs, nil
}
