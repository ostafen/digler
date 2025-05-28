package format_test

import (
	"bytes"
	"testing"

	"github.com/ostafen/diglet/internal/format"
	"github.com/stretchr/testify/require"
)

func TestReader_Reset(t *testing.T) {
	testData := []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz") // 62 bytes
	bufferSize := 10                                                                     // A small buffer size for testing edge cases easily

	r, err := format.NewReader(bytes.NewReader(testData), bufferSize)
	require.NoError(t, err)

	require.Equal(t, r.Bytes(), testData[:bufferSize])

	checkRead := func(off int64) int {
		nRead, err := r.Reset(off)
		require.NoError(t, err)
		require.Equal(t, r.Bytes(), testData[off:min(off+int64(bufferSize), int64(len(testData)))])

		return nRead
	}

	for i := range testData {
		nRead := checkRead(int64(i))
		if i == 0 || i >= len(testData)-bufferSize+1 {
			require.Equal(t, nRead, 0)
		} else {
			require.Equal(t, nRead, 1)
		}
	}

	require.Equal(t, r.Len(), 1)

	nRead := checkRead(int64(r.Offset()) - int64(bufferSize)/2)
	require.Equal(t, nRead, bufferSize/2)

	nRead = checkRead(int64(r.Offset()) - 2)
	require.Equal(t, nRead, 2)

	nRead = checkRead(int64(r.Offset()) - 4)
	require.Equal(t, nRead, 4)

	nRead = checkRead(12)
	require.Equal(t, nRead, bufferSize)

	nRead = checkRead(31)
	require.Equal(t, nRead, bufferSize)
}
