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
package api

import (
	"os"

	"github.com/ostafen/digler/pkg/sysinfo"
)

type SystemAPI struct {
	dataDir          string
	defaultOutputDir string
}

func NewSystemAPI(
	dataDir string,
	defaultOutputDir string,
) *SystemAPI {
	return &SystemAPI{
		dataDir:          dataDir,
		defaultOutputDir: defaultOutputDir,
	}
}

func (s *SystemAPI) WorkingDir() (string, error) {
	return os.Getwd()
}

func (s *SystemAPI) DataDir() string {
	return s.dataDir
}

func (s *SystemAPI) DefaultOutputDir() string {
	return s.defaultOutputDir
}

type DeviceInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Model string `json:"model"`
	Size  int64  `json:"size"`
}

func (s *SystemAPI) ListDevices() ([]DeviceInfo, error) {
	devices, err := sysinfo.ListDevices()
	if err != nil {
		return nil, err
	}

	deviceNames := make([]DeviceInfo, len(devices))
	for i, device := range devices {
		deviceNames[i] = DeviceInfo{
			Name:  device.Name,
			Path:  device.Path,
			Model: device.Model,
			Size:  device.Size,
		}
	}
	return deviceNames, nil
}
