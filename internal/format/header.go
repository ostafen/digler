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
package format

import (
	"fmt"
	"plugin"
)

type ScanResult struct {
	Name string
	Ext  string
	Size uint64
}

type FileHeader struct {
	Ext         string // File extension, e.g., "mp3", "wav"
	Description string
	Signatures  [][]byte
	ScanFile    func(r *Reader) (*ScanResult, error)
}

var fileHeaders = []FileHeader{
	// audio formats
	mp3FileHeader,
	wavFileHeader,
	sunAudioFileHeader,
	wmaFileHeader,
	// image formats
	jpegFileHeader,
	pngFileHeader,
	bmpFileHeader,
	gifFileHeader,
	pcxFileHeader,
	tiffFileHeader,
	// generic/documents formats
	zipFileHeader,
	rarFileHeader,
	pdfFileHeader,
	// database formats
	sqliteFileHeader,
}

func GetFileScanners(ext ...string) ([]FileScanner, error) {
	if len(ext) == 0 {
		scanners := GetAllFileScanners()
		return scanners, nil
	}

	headersByExt := make(map[string]FileHeader)
	for _, hdr := range fileHeaders {
		headersByExt[hdr.Ext] = hdr
	}

	scanners := make([]FileScanner, len(ext))
	for i, e := range ext {
		hdr, ok := headersByExt[e]
		if !ok {
			return nil, fmt.Errorf("unknown file extension: \"%s\"", hdr.Ext)
		}
		scanners[i] = &headerFileScanner{hdr: hdr}
	}
	return scanners, nil
}

func GetAllFileScanners() []FileScanner {
	scanners := make([]FileScanner, len(fileHeaders))
	for i, hdr := range fileHeaders {
		scanners[i] = &headerFileScanner{hdr: hdr}
	}
	return scanners
}

func BuildFileRegistry(scanners ...FileScanner) *FileRegistry {
	r := NewFileRegisty()
	for _, sc := range scanners {
		r.Add(sc)
	}
	return r
}

func LoadPlugins(pluginPaths ...string) ([]FileScanner, error) {
	scanners := make([]FileScanner, len(pluginPaths))
	for i, path := range pluginPaths {
		sc, err := loadPlugin(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin %s: %w", path, err)
		}
		scanners[i] = sc
	}
	return scanners, nil
}

func loadPlugin(path string) (FileScanner, error) {
	plug, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	symScanner, err := plug.Lookup("GetScanner")
	if err != nil {
		return nil, fmt.Errorf("plugin %s does not export FileScanner symbol: %w", path, err)
	}

	getScanner, ok := symScanner.(func() (FileScanner, error))
	if !ok {
		return nil, fmt.Errorf("plugin %s GetScanner has wrong type", path)
	}
	return getScanner()
}

func (r *FileRegistry) Signatures() int {
	return r.table.Size()
}
