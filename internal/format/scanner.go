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
	"bufio"
	"bytes"
	"fmt"
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
	Ext    string
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

				res, err := hdr.ScanFile(fileReader)
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
				return res.Size
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
