// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gif implements a GIF image decoder and encoder.
//
// The GIF specification is at https://www.w3.org/Graphics/GIF/spec-gif89a.txt.

// Copyright 2025 Stefano Scafiti. All rights reserved.
//
// This file implements a GIF decoder derived from the one in the Go standard library.
// It has been modified and extended specifically for file carving.
//
// Modifications are licensed under the MIT License; see the LICENSE file for details.
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

		// Stop at last chunk (mdat in Python version)
		if name == "mdat" {
			break
		}

		// Seek to next atom
		if _, err := r.Discard(int(size - 8)); err != nil { // subtract header size
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
