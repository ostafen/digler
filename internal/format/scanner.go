package format

import "fmt"

type Scanner struct {
	headers   []Header
	blockSize uint64
}

type FileInfo struct {
	Offset uint64 // Offset in the file where the format starts
	Size   uint64 // Size of the format in bytes
	Format string // Format type (e.g., "MP3", "WAV")
}

func NewScanner(blockSize uint64) *Scanner {
	return &Scanner{
		headers:   DefaultHeaders,
		blockSize: blockSize,
	}
}

func (sc *Scanner) AddHeader(header Header) {
	sc.headers = append(sc.headers, header)
}

func (sc *Scanner) Scan(data []byte) func(yield func(FileInfo) bool) {
	return func(yield func(FileInfo) bool) {
		for blockOffset := uint64(0); blockOffset < uint64(len(data)); {
			finfo, err := sc.scanFile(data, blockOffset)
			if err == nil {
				if !yield(finfo) {
					return
				}

				blockOffset = nextBlockOffset(finfo.Offset+finfo.Size, uint64(sc.blockSize))
			} else {
				blockOffset += uint64(sc.blockSize)
			}
		}
	}
}

func (sc *Scanner) scanFile(data []byte, startOffset uint64) (FileInfo, error) {
	for _, header := range sc.headers {
		endOffset, err := header.ScanFile(data[startOffset:])
		if err == nil {
			return FileInfo{
				Offset: startOffset,
				Size:   endOffset,
				Format: header.Ext,
			}, nil
		}
	}
	return FileInfo{}, fmt.Errorf("unable to scan file at offset %d", startOffset)
}

func nextBlockOffset(offset, blockSize uint64) uint64 {
	block := (offset + blockSize - 1) / blockSize
	return block * blockSize
}
