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

//go:build !windows
// +build !windows

package sysinfo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ListDevices lists block devices on a Linux system by reading from /sys/block.
// It returns a slice of DeviceInfo structs containing the device name, size, and model.
func ListDevices() ([]DeviceInfo, error) {
	const (
		blockPath = "/sys/block"
		devPath   = "/dev"
	)

	readFile := func(path string) (string, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}

	entries, err := os.ReadDir(blockPath)
	if err != nil {
		return nil, err
	}

	devices := make([]DeviceInfo, 0, len(entries))
	for _, entry := range entries {
		device := entry.Name()
		sizePath := filepath.Join(blockPath, device, "size")
		modelPath := filepath.Join(blockPath, device, "device/model")

		sectorsStr, _ := readFile(sizePath)
		sectors, err := strconv.ParseInt(sectorsStr, 10, 64)
		if err != nil {
			continue
		}
		model, err := readFile(modelPath)
		if err != nil {
			continue
		}

		devices = append(devices, DeviceInfo{
			Name:  device,
			Path:  filepath.Join(devPath, device),
			Size:  sectors * 512, // size is in 512-byte sectors
			Model: model,
		})
	}
	return devices, nil
}
