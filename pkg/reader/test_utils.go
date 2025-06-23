package reader

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"
)

// testReadSeeker performs randomized seek+read tests on a provided io.ReadSeeker constructor.
// - newReader: a function that returns a new io.ReadSeeker given the original buffer
// - original: the byte slice to validate against
// - trials: number of random seek+read operations to perform
func testReadSeeker(t *testing.T, newReader func([]byte) io.ReadSeeker) {
	const trials = 1000

	data := GenerateRandomBuffer(1024 * 10)
	rs := newReader(data)

	var buf [64]byte

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range trials {
		offset := rng.Intn(len(data))
		maxLen := len(data) - offset
		readLen := rng.Intn(64)
		if readLen > maxLen {
			readLen = maxLen
		}
		if readLen == 0 {
			readLen = 1
		}

		_, err := rs.Seek(int64(offset), io.SeekStart)
		if err != nil {
			t.Fatalf("trial %d: Seek(%d, SeekStart) failed: %v", i, offset, err)
		}

		n, err := rs.Read(buf[:readLen])
		if err != nil && err != io.EOF {
			t.Fatalf("trial %d: Read after Seek failed: %v", i, err)
		}

		expected := data[offset:]
		if len(expected) > readLen {
			expected = expected[:readLen]
		}

		if !bytes.Equal(buf[:n], expected) {
			t.Errorf("trial %d: mismatch at offset %d\nGot:      %v\nExpected: %v",
				i, offset, buf[:n], expected)
		}
	}
}

// GenerateRandomBuffer returns a random byte slice of the given size.
func GenerateRandomBuffer(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random data: " + err.Error())
	}
	return b
}
