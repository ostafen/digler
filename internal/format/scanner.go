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
package format

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ostafen/digler/internal/logger"
	"github.com/ostafen/digler/pkg/pbar"
	"github.com/ostafen/digler/pkg/reader"
)

type Scanner struct {
	blockSize   int
	maxFileSize uint64
	buf         []byte

	r         *FileRegistry
	logger    *logger.Logger
	bufReader *reader.BufferedReadSeeker
}

type FileInfo struct {
	Name   string
	Ext    string
	Offset uint64 // Offset in the file where the format starts
	Size   uint64 // Size of the format in bytes
}

func NewScanner(
	logger *logger.Logger,
	r *FileRegistry,
	bufferSize,
	blockSize int,
	maxFileSize uint64,
) *Scanner {
	return &Scanner{
		blockSize:   blockSize,
		maxFileSize: maxFileSize,
		buf:         make([]byte, roundToMul(bufferSize, int(blockSize))),
		r:           r,
		logger:      logger,
		bufReader:   reader.NewBufferedReadSeeker(nil, 4096),
	}
}

func (sc *Scanner) Scan(r io.ReaderAt, size uint64) func(yield func(FileInfo) bool) {
	return func(yield func(FileInfo) bool) {
		stop := false

		pb := pbar.NewProgressBarState(int64(size))
		defer pb.Finish()

		filesFound := 0

		for blockOffset := uint64(0); !stop && blockOffset < size; {
			n, err := r.ReadAt(sc.buf, int64(blockOffset))
			if err != nil && err != io.EOF {
				return
			}

			n = roundToMul(n, sc.blockSize) / sc.blockSize

			nextBlockOffset := blockOffset + uint64(len(sc.buf))

			sc.scanBuffer(n, func(blockIdx int, hdr FileHeader) uint64 {
				globalBlock := blockOffset/uint64(sc.blockSize) + uint64(blockIdx)
				globalOffset := globalBlock * uint64(sc.blockSize)

				pb.ProcessedBytes = int64(globalOffset)
				pb.FilesFound = filesFound
				pb.Render(false)

				bufData := sc.buf[blockIdx*sc.blockSize : n*sc.blockSize]

				remainingSize := max(
					int64(size)-(int64(blockOffset)+int64(len(sc.buf))),
					0,
				)

				mr := reader.NewMultiReadSeeker(
					[]io.ReadSeeker{
						bytes.NewReader(bufData),
						io.NewSectionReader(
							r,
							int64(blockOffset)+int64(len(sc.buf)),
							remainingSize,
						),
					},
					[]int64{int64(len(bufData)), remainingSize},
				)

				sc.bufReader.Reset(mr)

				maxSize := min(
					sc.maxFileSize,
					uint64(len(bufData))+uint64(remainingSize),
				)

				r := NewReader(
					sc.bufReader,
					maxSize,
				)

				res, err := hdr.ScanFile(r)
				if err != nil {
					return 0
				}

				finfo := scanResultToFileInfo(
					res,
					uint32(globalBlock),
					globalOffset,
					hdr.Ext,
				)

				stop = !yield(finfo)

				filesFound++

				nextBlockOffset = max(
					nextBlockOffset,
					roundToMul(globalOffset+res.Size, uint64(sc.blockSize)),
				)
				return res.Size
			})
			if err == io.EOF {
				break
			}
			blockOffset = nextBlockOffset
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

func roundToMul[T int | int64 | uint64](n, m T) T {
	k := (n + m - 1) / m
	return k * m
}

func scanResultToFileInfo(
	res *ScanResult,
	block uint32,
	offset uint64,
	defaultExt string,
) FileInfo {
	ext := defaultExt
	if res.Ext != "" {
		ext = res.Ext
	}

	name := res.Name
	if name == "" {
		name = fmt.Sprintf("f%d.%s", block, ext)
	}

	return FileInfo{
		Name:   name,
		Ext:    ext,
		Offset: offset,
		Size:   res.Size,
	}
}
