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
	"encoding/binary"
	"errors"
	"io"
)

var (
	CR3Magic  = []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'c', 'r', 'x'}
	CR3Marker = []byte("CanonCR3")
)

var cr3FileHeader = FileHeader{
	Ext:         "cr3",
	Description: "Canon CR3 RAW image format",
	Signatures:  [][]byte{CR3Magic},
	ScanFile:    ScanCr3,
}

// CR3 files are composed of a series of atoms (chunks) with 32-bit or 64-bit sizes.
// The scanner recognizes the CR3 magic header and the CanonCR3 marker to validate files.
// This implementation is adapted from the Python project: https://github.com/WojciechMula/recovercr3/blob/master/recovercr3.py
func ScanCr3(r *Reader) (*ScanResult, error) {
	// Check CR3 header
	magic := make([]byte, len(CR3Magic))
	_, err := r.Read(magic)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(magic, CR3Magic) {
		return nil, errors.New("not a CR3 file: missing magic")
	}

	// Seek to marker position (offset 64)
	if _, err := r.Discard(64 - len(CR3Magic)); err != nil {
		return nil, err
	}

	marker := make([]byte, len(CR3Marker))
	if _, err := io.ReadFull(r, marker); err != nil {
		return nil, err
	}
	if !bytes.Equal(marker, CR3Marker) {
		return nil, errors.New("not a CR3 file: missing CanonCR3 marker")
	}

	// Calculate total size by reading atoms
	var totalSize uint64
	for {
		name, size, err := readAtom(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// First atom must be 'ftyp'
		if totalSize == 0 && string(name) != "ftyp" {
			return nil, errors.New("first atom not 'ftyp'")
		}

		totalSize += size

		// Stop at last chunk
		if name == "mdat" {
			break
		}

		// Seek to next atom, subtract header size
		if _, err := r.Discard(int(size - 8)); err != nil {
			return nil, err
		}
	}

	return &ScanResult{
		Name: "0001",
		Ext:  "CR3",
		Size: totalSize,
	}, nil
}

func readAtom(r *Reader) (string, uint64, error) {
	var sizeBytes [4]byte
	if _, err := io.ReadFull(r, sizeBytes[:]); err != nil {
		return "", 0, err
	}

	name := make([]byte, 4)
	if _, err := io.ReadFull(r, name); err != nil {
		return "", 0, err
	}

	var size uint64

	size32 := binary.BigEndian.Uint32(sizeBytes[:])
	if size32 == 1 {
		// 64-bit size
		sizeBytes8 := make([]byte, 8)
		if _, err := io.ReadFull(r, sizeBytes8); err != nil {
			return "", 0, err
		}
		size = binary.BigEndian.Uint64(sizeBytes8)
	} else {
		size = uint64(size32)
	}
	return string(name), size, nil
}
