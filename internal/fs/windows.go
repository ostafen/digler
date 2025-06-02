//go:build windows
// +build windows

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
package fs

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type WindowsDiskFile struct {
	handle windows.Handle
	offset int64 // used for io.Reader
}

type diskFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	sys     any
}

func (fi *diskFileInfo) Name() string       { return fi.name }
func (fi *diskFileInfo) Size() int64        { return fi.size }
func (fi *diskFileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *diskFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *diskFileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi *diskFileInfo) Sys() interface{}   { return fi.sys }

// OpenWindowsDisk opens a disk/volume for raw reading
func Open(path string) (File, error) {
	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(path),
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0, // windows.FILE_FLAG_OVERLAPPED
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %w", path, err)
	}
	return &WindowsDiskFile{handle: handle}, nil
}

// Read reads from the current offset (for io.Reader)
func (d *WindowsDiskFile) Read(p []byte) (int, error) {
	var bytesRead uint32
	err := windows.ReadFile(d.handle, p, &bytesRead, nil)
	if err != nil {
		return int(bytesRead), err
	}
	d.offset += int64(bytesRead)
	return int(bytesRead), nil
}

func (d *WindowsDiskFile) ReadAt(p []byte, off int64) (int, error) {
	const sectorSize = 512

	// Calculate aligned offset and size
	alignedOffset := off / sectorSize * sectorSize
	alignmentDiff := int(off - alignedOffset)

	// Calculate aligned read size (must fully cover p)
	alignedSize := ((len(p) + alignmentDiff + sectorSize - 1) / sectorSize) * sectorSize

	// Allocate aligned buffer
	buf := make([]byte, alignedSize)

	// Prepare OVERLAPPED with aligned offset
	var bytesRead uint32
	ov := new(windows.Overlapped)
	ov.Offset = uint32(alignedOffset)
	ov.OffsetHigh = uint32(alignedOffset >> 32)

	err := windows.ReadFile(d.handle, buf, &bytesRead, ov)
	if err != nil {
		if err == syscall.ERROR_IO_PENDING {
			err = windows.GetOverlappedResult(d.handle, ov, &bytesRead, true)
		}
		if err != nil {
			return 0, fmt.Errorf("aligned read failed: %w", err)
		}
	}

	// Copy only requested portion
	n := copy(p, buf[alignmentDiff:])
	return n, nil
}

type DISK_GEOMETRY struct {
	Cylinders         int64
	MediaType         uint32
	TracksPerCylinder uint32
	SectorsPerTrack   uint32
	BytesPerSector    uint32
}

const IOCTL_DISK_GET_DRIVE_GEOMETRY = 0x70000

func (d *WindowsDiskFile) Stat() (os.FileInfo, error) {
	var geometry DISK_GEOMETRY
	var bytesReturned uint32

	err := windows.DeviceIoControl(
		d.handle,
		IOCTL_DISK_GET_DRIVE_GEOMETRY,
		nil,
		0,
		(*byte)(unsafe.Pointer(&geometry)),
		uint32(unsafe.Sizeof(geometry)),
		&bytesReturned,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("DeviceIoControl(IOCTL_DISK_GET_DRIVE_GEOMETRY) failed: %w", err)
	}

	size := geometry.Cylinders * int64(geometry.TracksPerCylinder) * int64(geometry.SectorsPerTrack) * int64(geometry.BytesPerSector)

	// Build a minimal FileInfo-like struct
	return &diskFileInfo{
		name:    "", // no name
		size:    size,
		mode:    0,
		modTime: time.Time{},
		sys:     geometry,
	}, nil
}

// Close closes the underlying handle
func (d *WindowsDiskFile) Close() error {
	return windows.CloseHandle(d.handle)
}
