package format

import (
	"fmt"
	"io"
)

type Reader struct {
	r io.ReaderAt

	off int64
	n   int
	buf []byte
}

func NewReader(r io.ReaderAt, bufSize int) (*Reader, error) {
	br := &Reader{
		r:   r,
		off: 0,
		n:   0,
		buf: make([]byte, bufSize),
	}

	n, err := r.ReadAt(br.buf, 0)
	if err != nil && err == io.EOF {
		return nil, err
	}
	br.buf = br.buf[:n]
	br.n = n

	return br, nil
}

// Reset repositions the reader's offset to 'off'.
// It attempts to reuse buffered data by shifting the buffer when possible,
// otherwise it reads new data from the underlying io.ReaderAt.
func (r *Reader) Reset(off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("invalid offset %d", off)
	}

	// If the buffer is empty, or the new offset is entirely outside the current buffer range,
	// or reading past the current buffer (forward jump too large).
	// In these cases, we must perform a full read.
	if off >= r.off+int64(r.n) || off+int64(len(r.buf)) <= r.off {
		nRead, err := r.r.ReadAt(r.buf, off)
		if err != nil && err != io.EOF {
			return 0, err
		}
		// If n is less than len(r.buf), it means we've read up to EOF or an error occurred
		// mid-read. We should adjust the effective buffer size.
		r.off = off
		r.n = nRead
		return nRead, nil
	}

	// off is within the current buffer or a small jump back/forward.

	// Scenario 1: Jumping forward within or just past the buffer.
	if off >= r.off {
		s := off - r.off
		if s > 0 {
			// Shift valid buffered content to the left by 's' locations.
			// The new effective length of already buffered data after shifting: r.n - s
			r.n = r.n - int(s)
			copy(r.buf, r.buf[s:s+int64(r.n)]) // copy `r.n` bytes from `s` onwards to start of buffer

			// Read new bytes into the end of the buffer.
			// We read into the remaining available space: len(r.buf) - r.n
			bytesToFill := len(r.buf) - r.n
			if bytesToFill > 0 {
				nRead, err := r.r.ReadAt(r.buf[r.n:], off+int64(r.n))
				if err != nil && err != io.EOF {
					return 0, err
				}

				r.n += nRead // Add the newly read bytes to the effective length
				r.off = off  // Update the reader's new offset
				return nRead, nil
			}
		}
		return 0, nil
	}

	// Scenario 2: Jumping backward. 'off' is less than 'r.off'.
	// We need to shift the buffer to the right and read new bytes at the beginning.
	s := r.off - off // How many bytes to shift right and read
	if s > 0 {
		// Shift 's' bytes to the right to make room for new bytes
		shiftRight(r.buf, r.n, int(s))

		_, err := r.r.ReadAt(r.buf[:s], off)
		if err == io.EOF {
			return 0, io.ErrUnexpectedEOF
		}
		if err != nil {
			return 0, err
		}

		r.off = off
		r.n = min(len(r.buf), r.n+int(s))
		r.off = off // Update the reader's new offset
		return int(s), nil
	}
	return 0, nil
}

func (r *Reader) Bytes() []byte {
	return r.buf[:r.n]
}

func (r *Reader) Offset() int {
	return int(r.off)
}

func (r *Reader) Len() int {
	return r.n
}

func shiftRight(buf []byte, n, s int) {
	for i := min(n+s-1, len(buf)-1); i >= s; i-- {
		buf[i] = buf[i-s]
	}
}
