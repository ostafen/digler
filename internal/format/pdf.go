package format

import (
	"bytes"
	"fmt"
)

var pdfFileHeader = FileHeader{
	Ext:        "pdf",
	Signatures: [][]byte{pdfHeader},
	ScanFile:   ScanPDF,
}

var (
	pdfHeader = []byte("%PDF-")
	eofMarker = []byte("%%EOF")

	pdfMaxFileSize = 16 * 1024 * 1024 // 16MB
)

// ScanPDF reads a byte stream from an io.Reader, identifies a potential PDF file,
// and returns its carved size.
//
// This function assumes that the entire potential PDF segment can be read into
// memory. For extremely large files (gigabytes), a more advanced streaming
// parser with limited look-behind would be necessary.
//
// It searches for the first occurrence of the standard PDF header (%PDF-X.Y)
// and the last occurrence of the end-of-file marker (%%EOF). The carved size
// is determined by the position of the last %%EOF marker plus its length.
//
// Parameters:
//
//	r io.Reader: The input stream from which to read the PDF data.
//
// Returns:
//
//	uint64: The size of the carved PDF file in bytes.
//	error: An error if the PDF header or EOF marker is not found, or if the
//	       EOF marker appears before the header.
func ScanPDF(r *Reader) (uint64, error) {
	var headerBuf [5]byte
	_, err := r.Read(headerBuf[:])
	if err != nil {
		return 0, err
	}

	if !bytes.Equal(headerBuf[:], pdfHeader) {
		return 0, fmt.Errorf("invalid pdf file")
	}

	var size uint64
	for {
		n := r.BytesRead()

		seeked, err := SeekAt(r, eofMarker, pdfMaxFileSize)
		if err != nil {
			return 0, err
		}
		if !seeked {
			break
		}

		_, err = r.Discard(len(eofMarker))
		if err != nil {
			return 0, err
		}

		size = r.BytesRead() - n + uint64(len(eofMarker))
	}

	if size == 0 {
		return 0, fmt.Errorf("invalid pdf file")
	}
	return size, nil
}
