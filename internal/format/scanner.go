package format

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"math"

	"github.com/ostafen/digler/pkg/pbar"
)

type Scanner struct {
	headers   []FileHeader
	blockSize int

	buf    []byte
	r      *FileRegistry
	logger *slog.Logger
}

type FileInfo struct {
	Name   string
	Format string
	Offset uint64 // Offset in the file where the format starts
	Size   uint64 // Size of the format in bytes
}

func NewScanner(logger *slog.Logger, r *FileRegistry, bufferSize, blockSize int) *Scanner {
	return &Scanner{
		headers:   fileHeaders,
		blockSize: blockSize,
		buf:       make([]byte, roundToMul(bufferSize, int(blockSize))),
		r:         r,
		logger:    logger,
	}
}

func (sc *Scanner) AddHeader(header FileHeader) {
	sc.headers = append(sc.headers, header)
}

func (sc *Scanner) Scan(r io.ReaderAt, size uint64) func(yield func(FileInfo) bool) {
	return func(yield func(FileInfo) bool) {
		stop := false

		pb := pbar.NewProgressBarState(int64(size))

		filesFound := 0

		for blockOffset := uint64(0); !stop && blockOffset < size; {
			n, err := r.ReadAt(sc.buf, int64(blockOffset))
			if err != nil && err != io.EOF {
				return
			}

			n = roundToMul(n, sc.blockSize) / sc.blockSize

			sc.scanBuffer(n, func(blockIdx int, hdr FileHeader) uint64 {
				globalBlock := blockOffset/uint64(sc.blockSize) + uint64(blockIdx)
				globalOffset := globalBlock * uint64(sc.blockSize)

				pb.ProcessedBytes = int64(globalOffset)
				pb.FilesFound = filesFound
				pb.Render(false)

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
					Offset: globalOffset,
					Size:   size,
					Format: hdr.Ext,
				}

				stop = !yield(finfo)

				filesFound++
				return size
			})
			if err == io.EOF {
				break
			}
			blockOffset += uint64(len(sc.buf))
		}

		pb.ProcessedBytes = int64(size)
		pb.FilesFound = filesFound
		pb.Render(true)
	}
}

func (sc *Scanner) scanBuffer(n int, scanFile func(blockIdx int, hdr FileHeader) uint64) {
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

func roundToMul(n, m int) int {
	k := (n + m - 1) / m
	return k * m
}
