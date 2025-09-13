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
package store

type RingBuffer[T any] struct {
	buf       []T
	nextIndex int
	n         int
}

func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		buf:       make([]T, capacity),
		nextIndex: 0,
		n:         0,
	}
}

func (buf *RingBuffer[T]) Append(x T) (T, bool) {
	old := buf.buf[buf.nextIndex]
	buf.buf[buf.nextIndex] = x
	buf.nextIndex = (buf.nextIndex + 1) % len(buf.buf)

	replaced := buf.n >= len(buf.buf)
	if !replaced {
		buf.n++
	}
	return old, replaced
}

func (buf *RingBuffer[T]) Get(i int) T {
	start := (buf.nextIndex - buf.n) + len(buf.buf)
	return buf.buf[(start+i)%len(buf.buf)]
}

func (buf *RingBuffer[T]) Reset() {
	buf.n = 0
	buf.nextIndex = 0
}

func (buf *RingBuffer[T]) Len() int {
	return buf.n
}
