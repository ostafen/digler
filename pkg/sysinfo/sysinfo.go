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
package sysinfo

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SysUnknown is a pre-defined SysInfo struct representing unknown system information.
var SysUnknown = SysInfo{
	Name:    runtime.GOOS,
	Release: "unknown",
	Version: "unknown",
}

// SysInfo holds the basic operating system details.
type SysInfo struct {
	Name    string // The name of the operating system (e.g., "linux", "darwin", "windows").
	Release string // The marketing name or release version of the OS (e.g., "Ubuntu", "macOS Sonoma", "Windows 11").
	Version string // The specific build or kernel version of the OS.
}

// Stat gathers and returns detailed operating system information.
// It uses runtime.GOOS to determine the current OS and then calls
// platform-specific functions to get more granular release and version data.
func Stat() (*SysInfo, error) {
	osSysname := runtime.GOOS
	osRelease := ""
	osVersion := ""

	switch osSysname {
	case "linux":
		osRelease, osVersion = getLinuxInfo()
	case "darwin":
		osRelease, osVersion = getDarwinInfo()
	case "windows":
		osRelease, osVersion = getWindowsInfo()
	default:
		osRelease, osVersion = "unknown", "unknown"
	}

	return &SysInfo{
		Name:    osSysname,
		Release: osRelease,
		Version: osVersion,
	}, nil
}

// getLinuxInfo retrieves OS release and version information for Linux systems.
// It attempts to read and parse the /etc/os-release file, which is a common
// standard for distributing OS identification data.
func getLinuxInfo() (string, string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "unknown", "unknown"
	}
	defer f.Close()

	var name, version string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "NAME=") {
			// Trim "NAME=" prefix and quotes from the value.
			name = strings.Trim(line[5:], `"`)
		}
		if strings.HasPrefix(line, "VERSION=") {
			// Trim "VERSION=" prefix and quotes from the value.
			version = strings.Trim(line[8:], `"`)
		}
	}
	return name, version
}

// getDarwinInfo retrieves OS release and version information for macOS systems.
// It executes the 'sw_vers' command and parses its output.
func getDarwinInfo() (string, string) {
	// Execute 'sw_vers' command to get macOS product information.
	output, err := exec.Command("sw_vers").Output()
	if err != nil {
		return "macOS", "unknown"
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var productName, productVersion string
	for scanner.Scan() {
		line := scanner.Text()
		// Look for "ProductName:" and "ProductVersion:" lines.
		if strings.HasPrefix(line, "ProductName:") {
			productName = strings.TrimSpace(strings.TrimPrefix(line, "ProductName:"))
		}
		if strings.HasPrefix(line, "ProductVersion:") {
			productVersion = strings.TrimSpace(strings.TrimPrefix(line, "ProductVersion:"))
		}
	}
	return productName, productVersion
}

// getWindowsInfo retrieves OS release and version information for Windows systems.
// It executes the 'cmd /c ver' command and returns the output.
func getWindowsInfo() (string, string) {
	// Execute 'cmd /c ver' to get the Windows version string.
	output, err := exec.Command("cmd", "/c", "ver").Output()
	if err != nil {
		return "Windows", "unknown"
	}
	version := strings.TrimSpace(string(output))
	return "Windows", version
}
