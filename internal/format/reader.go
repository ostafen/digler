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
	"io"

	"github.com/ostafen/digler/pkg/reader"
)

type Reader struct {
	r *reader.BufferedReadSeeker

	n    uint64
	size uint64
}

func NewReader(r *reader.BufferedReadSeeker, size uint64) *Reader {
	return &Reader{
		r:    r,
		size: size,
	}
}

func (r *Reader) ReadByte() (byte, error) {
	if r.n >= r.size {
		return 0, io.EOF
	}

	var buf [1]byte

	_, err := r.r.Read(buf[:])
	if err == nil {
		r.n++
	}
	return buf[0], err
}

func (r *Reader) Read(buf []byte) (int, error) {
	if r.n >= r.size {
		return 0, io.EOF
	}

	n, err := r.r.Read(buf)
	if err != nil && err != io.EOF {
		return n, err
	}

	if n > 0 {
		r.n += uint64(n)
	}
	return n, err
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	return r.r.Seek(offset, whence)
}

func (r *Reader) Unread(n int) error {
	_, err := r.Seek(-int64(n), io.SeekCurrent)
	return err
}

func (r *Reader) UnreadByte() error {
	return r.Unread(1)
}

func (r *Reader) Discard(n int) (int, error) {
	offset, err := r.r.Seek(int64(n), io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	prevOffset := offset - int64(n)
	if uint64(prevOffset) >= r.size {
		return 0, io.EOF
	}

	discarded := min(
		uint64(n),
		r.size-uint64(prevOffset),
	)

	r.n += discarded

	if discarded < uint64(n) {
		err = io.EOF
	}
	return int(discarded), err
}

func (r *Reader) Peek(n int) ([]byte, error) {
	return r.r.Peek(n)
}

func (r *Reader) BytesRead() uint64 {
	return r.n
}

func (r *Reader) BufferSize() int {
	return r.r.BufferSize()
}
