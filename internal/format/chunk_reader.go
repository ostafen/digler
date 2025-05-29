package format

import (
	"fmt"
	"io"
)

type ChunkBuffer struct {
	r io.ReaderAt

	startChunkSlot int
	startChunk     int

	numChunks int

	buf       []byte
	chunkSize int
}

func roundToMul(n, m int) int {
	k := (n + m - 1) / m
	return k * m
}

func NewChunkBuffer(r io.ReaderAt, size, chunkSize int) (*ChunkBuffer, error) {
	size = roundToMul(size, chunkSize)

	cb := &ChunkBuffer{
		r:              r,
		buf:            make([]byte, size),
		startChunk:     0,
		startChunkSlot: 0,
		numChunks:      -1,
		chunkSize:      chunkSize,
	}

	err := cb.loadFull(0)
	return cb, err
}

func (cb *ChunkBuffer) EnsureChunkIsBuffered(numChunk int) error {
	if numChunk < cb.startChunk {
		return fmt.Errorf("cannot load an older chunk")
	}

	var err error

	if cb.hasChunk(numChunk) {
		chunksToLoad := numChunk - cb.startChunk
		err = cb.advanceChunks(chunksToLoad)
	} else {
		err = cb.loadFull(numChunk)
	}
	return err
}

func (cb *ChunkBuffer) loadFull(startChunk int) error {
	chunkOff := startChunk * cb.chunkSize

	n, err := cb.r.ReadAt(cb.buf, int64(chunkOff))
	if err != nil && err != io.EOF {
		return err
	}

	cb.startChunk = startChunk
	cb.startChunkSlot = 0
	cb.numChunks = roundToMul(n, cb.chunkSize) / cb.chunkSize
	return nil
}

func (cb *ChunkBuffer) advanceChunks(n int) error {
	nChunksLoaded := 0
	for ; nChunksLoaded < n; nChunksLoaded++ {
		chunkOff := (cb.startChunk + cb.numChunks + nChunksLoaded) * cb.chunkSize

		bufOff := cb.startChunkSlot * cb.chunkSize
		_, err := cb.r.ReadAt(cb.buf[bufOff:bufOff+cb.chunkSize], int64(chunkOff))
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		cb.startChunkSlot = (cb.startChunkSlot + 1) % cb.MaxChunks()
	}

	cb.startChunk += nChunksLoaded
	cb.numChunks = cb.numChunks - n + nChunksLoaded

	return nil
}

func (cb *ChunkBuffer) hasChunk(numChunk int) bool {
	return numChunk >= cb.startChunk && numChunk < cb.startChunk+cb.numChunks
}

func (cb *ChunkBuffer) MaxChunks() int {
	return len(cb.buf) / cb.chunkSize
}
