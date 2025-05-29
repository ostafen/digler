package format

import (
	"bufio"
	"bytes"
	"io"
	"math"
)

type Scanner struct {
	headers   []FileHeader
	blockSize int

	currBlockOff int
	buf          []byte
	r            *FileRegistry
}

type FileInfo struct {
	Offset uint64 // Offset in the file where the format starts
	Size   uint64 // Size of the format in bytes
	Format string // Format type (e.g., "MP3", "WAV")
}

const DefaultBufferSize = 4 * 1024 * 1024

func NewScanner(blockSize int) *Scanner {
	return &Scanner{
		headers:   DefaultHeaders,
		blockSize: blockSize,
		buf:       make([]byte, roundToMul(DefaultBufferSize, int(blockSize))),
		r:         BuildRegistry(),
	}
}

func (sc *Scanner) AddHeader(header FileHeader) {
	sc.headers = append(sc.headers, header)
}

func (sc *Scanner) Scan(r io.ReaderAt, limit uint64) func(yield func(FileInfo) bool) {
	return func(yield func(FileInfo) bool) {

		// TODO: create a multi reader

		for blockOffset := uint64(0); blockOffset < limit; {
			n, err := r.ReadAt(sc.buf, int64(blockOffset))
			if err != nil && err == io.EOF {
				return
			}
			n = roundToMul(n, sc.blockSize) / sc.blockSize

			sc.currBlockOff = int(blockOffset)

			sc.scanBuffer(n, func(blockIdx int, hdr FileHeader) uint64 {
				fileReader := NewReader(
					bufio.NewReader(
						io.MultiReader(
							bytes.NewReader(sc.buf[blockIdx*sc.blockSize:n*sc.blockSize]),
							io.NewSectionReader(
								r,
								int64(blockOffset)+int64(len(sc.buf)),
								math.MaxInt64,
							),
						),
					),
				)

				size, err := hdr.ScanFile(fileReader)
				if err != nil {
					return 0
				}

				finfo := FileInfo{
					Offset: blockOffset + uint64(blockIdx)*uint64(sc.blockSize),
					Size:   size,
					Format: hdr.Ext,
				}

				yield(finfo)
				return size
			})
			if err == io.EOF {
				break
			}
			blockOffset += uint64(len(sc.buf))
		}
	}
}

func (sc *Scanner) scanBuffer(n int, scanFile func(blockIdx int, hdr FileHeader) uint64) {
	// TODO: use an adaptive search strategy depending on
	// how much matches you find in the blocks.
	// If only 1 match is found, which is highly likely, then it doesn't
	// make sense to cache older blocks in the chunk buffer as we will never roolback.

	// We assume each signature fits within a single block
	for blockIdx := 0; blockIdx < n; {
		var size uint64

		sc.r.Search(sc.buf[blockIdx*sc.blockSize:], func(hdr FileHeader) bool {
			size = scanFile(blockIdx, hdr)
			return size > 0
		})

		if size > 0 {
			fileBlocks := roundToMul(int(size), sc.blockSize) / sc.blockSize
			blockIdx += fileBlocks
		} else {
			blockIdx++
		}
	}
}
