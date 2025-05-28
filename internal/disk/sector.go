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
