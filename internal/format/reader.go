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
	"bufio"
	"io"
)

type Reader struct {
	buf *bufio.Reader

	n uint64
}

func NewReader(r *bufio.Reader) *Reader {
	return &Reader{
		buf: r,
	}
}

func (r *Reader) ReadByte() (byte, error) {
	b, err := r.buf.ReadByte()
	if err == nil {
		r.n++
	}
	return b, err
}

func (r *Reader) Read(buf []byte) (int, error) {
	n, err := r.buf.Read(buf)
	if err != nil && err != io.EOF {
		return n, err
	}

	if n > 0 {
		r.n += uint64(n)
	}
	return n, err
}

// TODO: this method is inefficient. Underlying reader must support Seek()
func (r *Reader) Discard(n int) (int, error) {
	copied, err := io.CopyN(io.Discard, r, int64(n))
	return int(copied), err
}

func (r *Reader) Peek(n int) ([]byte, error) {
	return r.buf.Peek(n)
}

func (r *Reader) BytesRead() uint64 {
	return r.n
}

func (r *Reader) BufferSize() int {
	return r.buf.Size()
}
