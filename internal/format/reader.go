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
