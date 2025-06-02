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
package disk

import (
	"encoding/binary"
	"fmt"
)

// MBRPartitionEntry represents a single 16-byte entry in the MBR's partition table.
// All multi-byte fields are stored as byte arrays to explicitly handle little-endian
// conversion when reading from the raw MBR byte slice.
type MBRPartitionEntry struct {
	BootIndicator uint8        // 0x00: 0x80 for bootable, 0x00 for inactive
	StartCHS      [3]byte      // 0x01: Starting Cylinder-Head-Sector address
	PartitionType MBRPartition // 0x04: Partition type ID (e.g., 0x0B for FAT32, 0x83 for Linux)
	EndCHS        [3]byte      // 0x05: Ending Cylinder-Head-Sector address
	StartLBA      [4]byte      // 0x08: Starting Logical Block Address (LBA) - uint32, Little-Endian
	TotalSectors  [4]byte      // 0x0C: Total sectors in partition - uint32, Little-Endian
}

// ReadStartLBA returns the starting LBA of the partition.
func (p *MBRPartitionEntry) ReadStartLBA() uint32 {
	return binary.LittleEndian.Uint32(p.StartLBA[:])
}

// ReadTotalSectors returns the total number of sectors in the partition.
func (p *MBRPartitionEntry) ReadTotalSectors() uint32 {
	return binary.LittleEndian.Uint32(p.TotalSectors[:])
}

// String provides a human-readable representation of an MBRPartitionEntry.
func (p *MBRPartitionEntry) String() string {
	bootable := "No"
	if p.BootIndicator == 0x80 {
		bootable = "Yes"
	}
	return fmt.Sprintf("  Bootable: %s (0x%02X)\n"+
		"  Partition Type: 0x%02X (%s)\n"+
		"  Start LBA: %d\n"+
		"  Total Sectors: %d\n"+
		"  Size: %d bytes (%s)",
		bootable, p.BootIndicator,
		p.PartitionType, getPartitionTypeName(p.PartitionType),
		p.ReadStartLBA(),
		p.ReadTotalSectors(),
		p.ReadTotalSectors()*512, // Assuming 512 bytes per sector
		formatBytes(uint64(p.ReadTotalSectors())*512))
}

// MBR represents the Master Boot Record structure.
type MBR struct {
	BootCode         [440]byte            // 0x000-0x1B7: Bootstrap code
	DiskSignature    [4]byte              // 0x1B8-0x1BB: Optional 32-bit disk signature
	Reserved         [2]byte              // 0x1BC-0x1BD: Usually 0x0000
	PartitionEntries [4]MBRPartitionEntry // 0x1BE-0x1FD: Four 16-byte partition entries
	Signature        [2]byte              // 0x1FE-0x1FF: MBR signature (0x55AA)
}

// ReadDiskSignature returns the disk signature as a uint32.
func (m *MBR) ReadDiskSignature() uint32 {
	return binary.LittleEndian.Uint32(m.DiskSignature[:])
}

// ReadSignature returns the MBR signature (should be 0xAA55).
func (m *MBR) ReadSignature() uint16 {
	return binary.LittleEndian.Uint16(m.Signature[:])
}

// String provides a human-readable representation of the MBR.
func (m *MBR) String() string {
	s := fmt.Sprintf("--- Master Boot Record (MBR) ---\n"+
		"Disk Signature: 0x%08X\n"+
		"MBR Signature: 0x%04X (Expected: 0xAA55)\n\n"+
		"--- Partition Table Entries ---",
		m.ReadDiskSignature(), m.ReadSignature())

	for i, entry := range m.PartitionEntries {
		s += fmt.Sprintf("\nPartition %d:\n%s", i+1, entry.String())
	}
	return s
}

// ParseMBR parses a 512-byte slice into an MBR struct.
// It assumes the input slice is exactly 512 bytes long and contains
// the raw binary data of an MBR in little-endian format.
func ParseMBR(data []byte) (*MBR, error) {
	const MBR_SIZE = 512
	const MBR_SIGNATURE_OFFSET = 0x1FE

	if len(data) != MBR_SIZE {
		return nil, fmt.Errorf("input data slice size mismatch: expected %d bytes, got %d bytes", MBR_SIZE, len(data))
	}

	var mbr MBR

	// Copy bootstrap code
	copy(mbr.BootCode[:], data[0x000:0x1B8])
	// Copy disk signature
	copy(mbr.DiskSignature[:], data[0x1B8:0x1BC])
	// Copy reserved bytes
	copy(mbr.Reserved[:], data[0x1BC:0x1BE])

	// Populate partition entries
	for i := 0; i < 4; i++ {
		entryOffset := 0x1BE + (i * 16) // Each entry is 16 bytes
		entryBytes := data[entryOffset : entryOffset+16]

		mbr.PartitionEntries[i].BootIndicator = entryBytes[0x00]
		copy(mbr.PartitionEntries[i].StartCHS[:], entryBytes[0x01:0x04])
		mbr.PartitionEntries[i].PartitionType = MBRPartition(entryBytes[0x04])
		copy(mbr.PartitionEntries[i].EndCHS[:], entryBytes[0x05:0x08])
		copy(mbr.PartitionEntries[i].StartLBA[:], entryBytes[0x08:0x0C])
		copy(mbr.PartitionEntries[i].TotalSectors[:], entryBytes[0x0C:0x10])
	}

	// Copy MBR signature
	copy(mbr.Signature[:], data[MBR_SIGNATURE_OFFSET:MBR_SIGNATURE_OFFSET+2])

	// Validate MBR signature
	if mbr.ReadSignature() != 0xAA55 {
		return nil, fmt.Errorf("invalid MBR signature: expected 0xAA55, got 0x%04X", mbr.ReadSignature())
	}
	return &mbr, nil
}

type MBRPartition uint8

const (
	PartitionTypeEmpty MBRPartition = iota
	PartitionTypeFAT12
	PartitionTypeXENIXRoot
	PartitionTypeXENIXUsr
	PartitionTypeFAT16LessThan32MB
	PartitionTypeExtendedCHS
	PartitionTypeFAT16GreaterThan32MB
	PartitionTypeNTFSHPFSexFATQNX
	PartitionTypeAIX
	PartitionTypeAIXBootable
	PartitionTypeOs2BootManager
	PartitionTypeFAT32CHS
	PartitionTypeFAT32LBA
	PartitionTypeFAT16LBA
	PartitionTypeUnknown
	PartitionTypeExtendedLBA
	PartitionTypeLinuxSwap
	PartitionTypeLinuxFilesystem
	PartitionTypeGPTProtectiveMBR
	PartitionTypeEFISystemPartition
	PartitionTypeGPT = 0xEE
)

// Helper function to map common partition type IDs to names
func getPartitionTypeName(id MBRPartition) string {
	switch id {
	case PartitionTypeEmpty:
		return "Empty"
	case PartitionTypeFAT12:
		return "FAT12"
	case PartitionTypeFAT16LessThan32MB:
		return "FAT16 (<32MB)"
	case PartitionTypeExtendedCHS:
		return "Extended (CHS)"
	case PartitionTypeFAT16GreaterThan32MB:
		return "FAT16 (>32MB)"
	case PartitionTypeNTFSHPFSexFATQNX:
		return "NTFS/HPFS/exFAT/QNX"
	case PartitionTypeFAT32CHS:
		return "FAT32 (CHS)"
	case PartitionTypeFAT32LBA:
		return "FAT32 (LBA)"
	case PartitionTypeFAT16LBA:
		return "FAT16 (LBA)"
	case PartitionTypeExtendedLBA:
		return "Extended (LBA)"
	case PartitionTypeLinuxSwap:
		return "Linux swap"
	case PartitionTypeLinuxFilesystem:
		return "Linux filesystem"
	case PartitionTypeGPTProtectiveMBR:
		return "GPT Protective MBR"
	case PartitionTypeEFISystemPartition:
		return "EFI System Partition"
	default:
		return "Unknown"
	}
}

func formatBytes(b uint64) string {
	const (
		_  = iota             // 0
		KB = 1 << (10 * iota) // 1 << 10 (1024)
		MB = 1 << (10 * iota) // 1 << 20 (1024 * 1024)
		GB = 1 << (10 * iota) // 1 << 30 (1024 * 1024 * 1024)
		TB = 1 << (10 * iota) // 1 << 40 (1024 * 1024 * 1024 * 1024)
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.2f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(KB))
	default: // Handle bytes less than 1KB
		return fmt.Sprintf("%d B", b)
	}
}
