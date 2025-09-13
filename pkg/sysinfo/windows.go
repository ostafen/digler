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

//go:build windows
// +build windows

package sysinfo

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	getVolumeInformation = kernel32.NewProc("GetVolumeInformationW")
	getDriveType         = kernel32.NewProc("GetDriveTypeW")
)

func ListDevices() ([]DeviceInfo, error) {
	drivesBitmask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, err
	}

	devices := make([]DeviceInfo, 0, 26)
	for i := 0; i < 26; i++ {
		if drivesBitmask&(1<<i) == 0 {
			continue
		}

		drive := string('A'+i) + `:\`

		// Get volume label
		volName := make([]uint16, syscall.MAX_PATH+1)
		ret, _, _ := getVolumeInformation.Call(
			uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(drive))),
			uintptr(unsafe.Pointer(&volName[0])),
			uintptr(len(volName)),
			0, 0, 0, 0, 0,
		)
		label := ""
		if ret != 0 {
			label = syscall.UTF16ToString(volName)
		}

		// Get drive type
		ret, _, _ = getDriveType.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(drive))))
		removable := ret == 2 // DRIVE_REMOVABLE

		// Get total size
		var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes int64
		err := windows.GetDiskFreeSpaceEx(syscall.StringToUTF16Ptr(drive),
			(*uint64)(unsafe.Pointer(&freeBytesAvailable)),
			(*uint64)(unsafe.Pointer(&totalNumberOfBytes)),
			(*uint64)(unsafe.Pointer(&totalNumberOfFreeBytes)),
		)
		if err != nil {
			continue
		}

		devices = append(devices, DeviceInfo{
			Name:      label,
			Path:      drive,
			Size:      totalNumberOfBytes,
			Removable: removable,
		})
	}
	return devices, nil
}
