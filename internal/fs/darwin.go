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

//go:build darwin
// +build darwin

package fs

import (
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

const defaultSectorSize = 512

// dkiocGetBlockSize is DKIOCGETBLOCKSIZE from <sys/disk.h>: _IOR('d', 24, uint32_t).
// golang.org/x/sys/unix does not export it.
const dkiocGetBlockSize = 0x40046418

// alignedFile wraps a raw character device (/dev/rdiskN). The kernel rejects
// raw-device reads whose offset or length is not a multiple of the device
// sector size (EINVAL), so ReadAt rounds the request out to sector boundaries
// and copies back only the bytes the caller asked for. Only ReadAt is
// aligned: sequential Read/Seek come from the embedded *os.File, and no
// caller uses them on raw devices today.
type alignedFile struct {
	*os.File
	sectorSize int64
}

// Open opens a file or device. Raw character devices get sector-aligned reads.
func Open(path string) (File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return f, nil
	}

	sectorSize := int64(defaultSectorSize)
	// DKIOCGETBLOCKSIZE writes a uint32; IoctlGetInt reads it into a zeroed
	// int, which is correct on little-endian darwin. x/sys has no darwin
	// IoctlGetUint32.
	if bs, err := unix.IoctlGetInt(int(f.Fd()), dkiocGetBlockSize); err == nil && bs > 0 {
		sectorSize = int64(bs)
	}
	return &alignedFile{File: f, sectorSize: sectorSize}, nil
}

func (a *alignedFile) ReadAt(p []byte, off int64) (int, error) {
	return alignedReadAt(a.File, a.sectorSize, p, off)
}

func alignedReadAt(r io.ReaderAt, ss int64, p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("fs: negative offset")
	}
	if len(p) == 0 {
		return 0, nil
	}

	// Fast path: already aligned.
	if off%ss == 0 && int64(len(p))%ss == 0 {
		return r.ReadAt(p, off)
	}

	start := off / ss * ss
	end := (off + int64(len(p)) + ss - 1) / ss * ss
	buf := make([]byte, end-start)

	n, err := r.ReadAt(buf, start)
	skip := int(off - start)
	if n <= skip {
		if err == nil {
			err = io.EOF
		}
		return 0, err
	}

	got := copy(p, buf[skip:n])
	if got < len(p) {
		if err == nil {
			err = io.ErrUnexpectedEOF
		}
		return got, err
	}
	// Full request satisfied; a trailing EOF on the padded read is irrelevant.
	return got, nil
}
