package mmap

import (
	"fmt"
	"os"
	"syscall"
)

// MmapFile represents a memory-mapped file region.
type MmapFile struct {
	Data         []byte   // The memory-mapped byte slice
	File         *os.File // The underlying opened file
	FileSize     int      // Total size of the underlying file
	MappedOffset int      // The starting offset of the mapped region within the file
	MappedLength int      // The length of the mapped region
}

func NewMmapFile(
	filePath string,
) (*MmapFile, error) {
	return NewMmapFileRegion(filePath, 0, 0)
}

// NewMmapFileRegion creates a new memory-mapped region from a file.
//
// filePath: The path to the file or raw disk device (e.g., "/dev/sda").
// offset:   The starting byte offset within the file to map. Must be page-aligned.
// length:   The number of bytes to map from the file, starting at `offset`.
//
//	If `length` is 0, the mapping will extend from `offset` to the end of the file.
//
// If mapping a raw disk device, ensure the path is correct and the program has root privileges.
func NewMmapFileRegion(
	filePath string,
	offset int,
	length int,
) (*MmapFile, error) {
	// Open the file/device
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", filePath, err)
	}

	// Get file/device size
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to get file info for %q: %w", filePath, err)
	}
	fileSize := int(fi.Size())

	if fileSize == 0 {
		f.Close()
		return nil, fmt.Errorf("file %q is empty, cannot mmap", filePath)
	}

	// Validate offset and length
	if offset < 0 {
		f.Close()
		return nil, fmt.Errorf("offset cannot be negative: %d", offset)
	}
	if offset >= fileSize {
		f.Close()
		return nil, fmt.Errorf("offset %d is beyond file size %d", offset, fileSize)
	}

	// If length is 0, map from offset to the end of the file
	actualMappedLength := length
	if length == 0 {
		actualMappedLength = fileSize - offset
	}

	if offset+actualMappedLength > fileSize {
		f.Close()
		return nil, fmt.Errorf("requested mapping (offset %d + length %d) extends beyond file size %d", offset, actualMappedLength, fileSize)
	}
	if actualMappedLength <= 0 {
		f.Close()
		return nil, fmt.Errorf("calculated mapped length is zero or negative: %d", actualMappedLength)
	}

	// Ensure offset is page-aligned for mmap.
	// syscall.Getpagesize() returns the system's memory page size.
	pageSize := syscall.Getpagesize()
	if offset%pageSize != 0 {
		f.Close()
		return nil, fmt.Errorf("offset %d is not page-aligned (page size: %d)", offset, pageSize)
	}

	// Perform the mmap operation
	// PROT_READ: pages may be read.
	// MAP_SHARED: updates to the mapping are visible to other processes mapping the same file,
	//             and are carried through to the underlying file.
	data, err := syscall.Mmap(
		int(f.Fd()),        // File descriptor
		int64(offset),      // Offset within the file to start mapping
		actualMappedLength, // Length of the mapping
		syscall.PROT_READ,  // Read protection
		syscall.MAP_SHARED, // Shared mapping
	)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to mmap file %q at offset %d with length %d: %w", filePath, offset, actualMappedLength, err)
	}

	return &MmapFile{
		Data:         data,
		File:         f,
		FileSize:     fileSize,
		MappedOffset: offset,
		MappedLength: actualMappedLength,
	}, nil
}

// Close unmaps the memory region and closes the underlying file.
func (mr *MmapFile) Close() error {
	var err error
	if mr.Data != nil {
		err = syscall.Munmap(mr.Data)
		if err != nil {
			return fmt.Errorf("failed to munmap: %w", err)
		}
		mr.Data = nil // Clear the reference to the unmapped memory
	}

	if mr.File != nil {
		closeErr := mr.File.Close()
		if closeErr != nil {
			if err != nil { // If munmap also failed, return a combined error
				return fmt.Errorf("failed to munmap (%w) and close file (%v)", err, closeErr)
			}
			return fmt.Errorf("failed to close file: %w", closeErr)
		}
		mr.File = nil
	}
	return nil
}
