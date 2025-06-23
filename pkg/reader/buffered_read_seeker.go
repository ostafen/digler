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
package reader

import (
	"errors"
	"fmt"
	"io"
)

type BufferedReadSeeker struct {
	src     io.ReadSeeker
	buf     []byte
	currPos int64 // global read offset
	off     int   // read offset in buffer
	size    int   // number of valid bytes in buffer
}

func NewBufferedReadSeeker(src io.ReadSeeker, bufSize int) *BufferedReadSeeker {
	return &BufferedReadSeeker{
		src:     src,
		buf:     make([]byte, bufSize),
		currPos: 0,
		off:     0,
		size:    0,
	}
}

func (b *BufferedReadSeeker) fillBuffer() error {
	// slide existing data to the beginning of the buffer
	copied := copy(b.buf, b.buf[b.off:b.size])

	n, err := b.src.Read(b.buf[copied:])
	if err != nil && err != io.EOF {
		return err
	}
	b.size = n + copied
	b.currPos += int64(b.off)
	b.off = 0
	return nil
}

func (b *BufferedReadSeeker) Read(p []byte) (int, error) {
	readBytes := 0
	for readBytes < len(p) {
		if b.off >= b.size {
			if err := b.fillBuffer(); err != nil {
				return 0, err
			}
			if b.size == 0 {
				return readBytes, io.EOF
			}
		}
		n := copy(p[readBytes:], b.buf[b.off:b.size])
		b.off += n
		readBytes += n
	}
	return readBytes, nil
}

func (b *BufferedReadSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart, io.SeekEnd:
	case io.SeekCurrent:
		offset += b.currPos + int64(b.off)
		whence = io.SeekStart
	default:
		return -1, fmt.Errorf("BufferedReadSeeker.Seek(): invalid whence: %d", whence)
	}

	if offset < 0 {
		return -1, fmt.Errorf("BufferedReadSeeker.Seek: negative position")
	}

	// If the new offset falls within the buffer's current data range,
	// adjust b.off to reflect the new starting position within that data.
	if offset >= b.currPos && offset < b.currPos+int64(b.size) {
		shift := offset - (b.currPos + int64(b.off))
		b.off += int(shift)
		return offset, nil
	}

	newOffset, err := b.src.Seek(offset, whence)
	if err != nil {
		return 0, err
	}

	// Discard any buffered data and reset the position
	b.off = 0
	b.size = 0
	b.currPos = newOffset
	return newOffset, nil
}

func (b *BufferedReadSeeker) Peek(n int) ([]byte, error) {
	if n > len(b.buf) {
		return nil, errors.New("peek size exceeds buffer capacity")
	}

	// Fill the buffer if there's not enough data available
	if b.off+n > b.size {
		if err := b.fillBuffer(); err != nil {
			return nil, err
		}
	}

	available := b.size - b.off
	if n > available {
		return b.buf[b.off:b.size], io.EOF
	}
	return b.buf[b.off : b.off+n], nil
}

func (b *BufferedReadSeeker) Reset(r io.ReadSeeker) {
	b.src = r
	b.off = 0
	b.size = 0
	b.currPos = 0
}

func (b *BufferedReadSeeker) BufferSize() int {
	return len(b.buf)
}
