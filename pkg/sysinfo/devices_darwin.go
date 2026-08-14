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

//go:build darwin
// +build darwin

package sysinfo

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"howett.net/plist"
)

type diskutilList struct {
	AllDisks              []string           `plist:"AllDisks"`
	AllDisksAndPartitions []diskutilDiskInfo `plist:"AllDisksAndPartitions"`
}

type diskutilDiskInfo struct {
	DeviceIdentifier string             `plist:"DeviceIdentifier"`
	Size             int64              `plist:"Size"`
	Content          string             `plist:"Content"`
	VolumeName       string             `plist:"VolumeName"`
	MountPoint       string             `plist:"MountPoint"`
	OSInternal       bool               `plist:"OSInternal"`
	RemovableMedia   bool               `plist:"RemovableMedia"`
	Partitions       []diskutilDiskInfo `plist:"Partitions"`
	APFSVolumes      []diskutilDiskInfo `plist:"APFSVolumes"`
}

func isSystemNoise(name, content, mountPoint string) bool {
	// Filter out internal macOS APFS helper volumes and CoreSimulator mounts
	systemNames := []string{
		"iscpreboot", "xart", "hardware", "preboot", "recovery", "update", "vm",
		"apple_apfs_isc", "apple_apfs_recovery", "efi",
	}
	nameLower := strings.ToLower(name)
	contentLower := strings.ToLower(content)

	for _, sys := range systemNames {
		if nameLower == sys || contentLower == sys {
			return true
		}
	}

	if strings.Contains(mountPoint, "CoreSimulator") || strings.Contains(nameLower, "simulator") {
		return true
	}

	return false
}

// ListDevices lists storage devices and volumes on macOS using 'diskutil list -plist'.
func ListDevices() ([]DeviceInfo, error) {
	cmd := exec.Command("diskutil", "list", "-plist")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run diskutil: %w", err)
	}

	var parsed diskutilList
	decoder := plist.NewDecoder(bytes.NewReader(out))
	if err := decoder.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("failed to parse diskutil plist: %w", err)
	}

	devices := make([]DeviceInfo, 0)
	seen := make(map[string]bool)

	addDev := func(devID, volumeName, content, mountPoint string, size int64, removable bool, isWholeDisk bool) {
		if devID == "" || size <= 0 || seen[devID] {
			return
		}

		if !isWholeDisk && isSystemNoise(volumeName, content, mountPoint) {
			return
		}

		seen[devID] = true

		displayName := volumeName
		if displayName == "" {
			displayName = devID
		} else if isWholeDisk {
			displayName = fmt.Sprintf("%s (%s)", devID, volumeName)
		} else {
			displayName = fmt.Sprintf("%s (%s)", volumeName, devID)
		}

		model := content
		if model == "" {
			if isWholeDisk {
				model = "Physical Disk"
			} else {
				model = "Partition"
			}
		}

		// Use /dev/rdiskX (raw character device) to prevent EBUSY locks on mounted volumes and maximize read speed
		rawPath := "/dev/r" + devID

		devices = append(devices, DeviceInfo{
			Name:      displayName,
			Path:      rawPath,
			Size:      size,
			Model:     model,
			Removable: removable,
		})
	}

	for _, disk := range parsed.AllDisksAndPartitions {
		// Whole disk
		addDev(disk.DeviceIdentifier, disk.VolumeName, disk.Content, disk.MountPoint, disk.Size, disk.RemovableMedia, true)

		// Partitions
		for _, part := range disk.Partitions {
			addDev(part.DeviceIdentifier, part.VolumeName, part.Content, part.MountPoint, part.Size, disk.RemovableMedia, false)
		}

		// APFS Volumes
		for _, apfs := range disk.APFSVolumes {
			addDev(apfs.DeviceIdentifier, apfs.VolumeName, apfs.Content, apfs.MountPoint, apfs.Size, disk.RemovableMedia, false)
		}
	}

	return devices, nil
}
