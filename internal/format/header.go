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

import "fmt"

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
	pdfFileHeader,
}

func FileHeaders(ext ...string) ([]FileHeader, error) {
	if len(ext) == 0 {
		return fileHeaders, nil
	}

	headersByExt := make(map[string]FileHeader)
	for _, hdr := range fileHeaders {
		headersByExt[hdr.Ext] = hdr
	}

	headers := make([]FileHeader, len(ext))
	for i, e := range ext {
		hdr, ok := headersByExt[e]
		if !ok {
			return nil, fmt.Errorf("unknown file extension: \"%s\"", hdr.Ext)
		}
		headers[i] = hdr
	}
	return headers, nil
}

func BuildFileRegistry(headers ...FileHeader) *FileRegistry {
	r := NewFileRegisty()
	for _, hdr := range headers {
		r.Add(hdr)
	}
	return r
}

func (r *FileRegistry) Signatures() int {
	return r.table.Size()
}
