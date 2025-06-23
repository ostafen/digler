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
	"fmt"
	"io"
	"sort"
)

type MultiReadSeeker struct {
	readers  []io.ReadSeeker
	cumSizes []int64

	currReader int
	currOff    int64
	size       int64 // total size
}

func NewMultiReadSeeker(
	readers []io.ReadSeeker,
	sizes []int64,
) *MultiReadSeeker {
	size := int64(0)
	for _, s := range sizes {
		size += s
	}

	for i := range sizes[:len(sizes)-1] {
		sizes[i+1] += sizes[i]
	}

	return &MultiReadSeeker{
		currReader: -1,
		currOff:    0,
		readers:    readers,
		cumSizes:   sizes,
		size:       size,
	}
}

func (r *MultiReadSeeker) Read(buf []byte) (int, error) {
	if r.currReader < 0 {
		if err := r.advanceReader(); err != nil {
			return 0, err
		}
	}

	bytesRead := 0
	for bytesRead < len(buf) && r.currReader < len(r.readers) {
		n, err := r.readers[r.currReader].Read(buf[bytesRead:])
		if err != nil && err != io.EOF {
			return bytesRead, err
		}

		bytesRead += n
		r.currOff += int64(n)
		if err == io.EOF {
			if r.currReader+1 == len(r.readers) {
				break
			}

			if err := r.advanceReader(); err != nil {
				return bytesRead, err
			}
		}
	}

	if bytesRead < len(buf) {
		return bytesRead, io.EOF
	}
	return bytesRead, nil
}

func (r *MultiReadSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += r.currOff
	case io.SeekEnd:
		offset = r.size + offset
	default:
		return -1, fmt.Errorf("MultiReadSeeker.Seek: invalid whence (%d)", whence)
	}

	if offset < 0 {
		return -1, fmt.Errorf("MultiReadSeeker.Seek: negative position")
	}

	if offset > int64(r.size) {
		r.currOff = offset
		r.currReader = len(r.readers)
		return offset, nil
	}

	i := sort.Search(len(r.readers), func(i int) bool {
		return r.cumSizes[i] > offset
	})
	r.currReader = i

	if i < len(r.readers) {
		var base int64
		if i > 0 {
			base = r.cumSizes[i-1]
		}

		if _, err := r.readers[r.currReader].Seek(offset-base, io.SeekStart); err != nil {
			return -1, err
		}
	}

	r.currOff = offset
	return offset, nil
}

func (r *MultiReadSeeker) advanceReader() error {
	i := r.currReader + 1

	if _, err := r.readers[i].Seek(0, io.SeekStart); err != nil {
		return err
	}
	r.currReader = i
	return nil
}
