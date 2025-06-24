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
	"bytes"
	"fmt"
)

var pdfFileHeader = FileHeader{
	Ext:         "pdf",
	Description: "Portable Document Format",
	Signatures:  [][]byte{pdfHeader},
	ScanFile:    ScanPDF,
}

var (
	pdfHeader = []byte("%PDF-")
	eofMarker = []byte("%%EOF")

	pdfMaxFileSize = 16 * 1024 * 1024 // 16MB
)

// ScanPDF reads a byte stream from an io.Reader, identifies a potential PDF file,
// and returns its carved size.
//
// It searches for the first occurrence of the standard PDF header (%PDF-X.Y)
// and the last occurrence of the end-of-file marker (%%EOF). The carved size
// is determined by the position of the last %%EOF marker plus its length.
func ScanPDF(r *Reader) (*ScanResult, error) {
	var headerBuf [5]byte
	_, err := r.Read(headerBuf[:])
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(headerBuf[:], pdfHeader) {
		return nil, fmt.Errorf("invalid pdf file")
	}

	var size uint64
	for {
		n := r.BytesRead()

		seeked, err := SeekAt(r, eofMarker, pdfMaxFileSize)
		if err != nil {
			return nil, err
		}
		if !seeked {
			break
		}

		_, err = r.Discard(len(eofMarker))
		if err != nil {
			return nil, err
		}

		size = r.BytesRead() - n + uint64(len(eofMarker))
	}

	if size == 0 {
		return nil, fmt.Errorf("invalid pdf file")
	}
	return &ScanResult{Size: size}, nil
}
