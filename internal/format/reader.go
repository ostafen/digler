package format

import (
	"bufio"
	"io"
)

type reader interface {
	io.Reader
	io.ByteReader
}

type Reader struct {
	r *bufio.Reader

	n int
}

func NewReader(r *bufio.Reader) *Reader {
	return &Reader{r: r}
}

func (r *Reader) ReadByte() (byte, error) {
	b, err := r.r.ReadByte()
	if err == nil {
		r.n++
	}
	return b, err
}

func (r *Reader) Read(buf []byte) (int, error) {
	n, err := r.r.Read(buf)
	if err != nil && err != io.EOF {
		return n, err
	}

	if n > 0 {
		r.n += n
	}
	return n, err
}

func (r *Reader) Discard(n int) error {
	_, err := io.CopyN(io.Discard, r.r, int64(n))
	return err
}

func (r *Reader) Peek(n int) ([]byte, error) {
	return r.r.Peek(n)
}
