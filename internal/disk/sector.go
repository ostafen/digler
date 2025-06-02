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
package disk

const DefaultBlocksize = 512

func GuessBlockSize(fileOffsets []uint64) (uint64, uint64) {
	if len(fileOffsets) == 0 {
		return DefaultBlocksize, 0
	}

	var blockSize uint64 = 128 * 512 // Start with 64KB
	offset := fileOffsets[0] % uint64(blockSize)

	for valid := false; !valid; {
		blockSize, offset, valid = EnforceAlignment(fileOffsets, blockSize, offset)
	}
	return blockSize, offset
}

func EnforceAlignment(offsets []uint64, blockSize, offset uint64) (uint64, uint64, bool) {
	for _, off := range offsets {
		if off%uint64(blockSize) != offset && blockSize > DefaultBlocksize {
			return blockSize >> 1, off % uint64(blockSize), false
		}
	}
	return blockSize, offset, true
}
