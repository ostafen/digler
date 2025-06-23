package reader

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
)

func TestMultiReadSeekerRandomSeek(t *testing.T) {
	testReadSeeker(t, func(data []byte) io.ReadSeeker {
		n := len(data)

		var (
			readers []io.ReadSeeker
			sizes   []int64
		)

		size := 0
		for size < n {
			sz := min(
				rand.Intn(1024)+1,
				n-size,
			)

			chunk := data[size : size+sz]
			readers = append(readers, bytes.NewReader(chunk))

			sizes = append(sizes, int64(sz))
			size += sz
		}
		return NewMultiReadSeeker(readers, sizes)
	})
}

func TestBufferedSeeker(t *testing.T) {
	testReadSeeker(t, func(data []byte) io.ReadSeeker {
		return NewBufferedReadSeeker(bytes.NewReader(data), 4096)
	})
}
