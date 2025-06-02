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

type ScanResult struct {
	Name string
	Ext  string
	Size uint64
}

type FileHeader struct {
	Ext        string // File extension, e.g., "mp3", "wav"
	Signatures [][]byte
	ScanFile   func(r *Reader) (*ScanResult, error)
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
	// generic/documents formats
	zipFileHeader,
	pdfFileHeader,
}

func BuildFileRegistry() *FileRegistry {
	r := NewFileRegisty()
	for _, hdr := range fileHeaders {
		r.Add(hdr)
	}
	return r
}

func (r *FileRegistry) Formats() []string {
	formats := make([]string, len(fileHeaders))
	for i := range formats {
		formats[i] = fileHeaders[i].Ext
	}
	return formats
}

func (r *FileRegistry) Signatures() int {
	n := 0
	for _, hdr := range fileHeaders {
		n += len(hdr.Signatures)
	}
	return n
}
