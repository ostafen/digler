package format

import (
	"math"

	art "github.com/plar/go-adaptive-radix-tree/v2"
)

type FileRegistry struct {
	tree      art.Tree
	minKeyLen int
}

type headers []FileHeader

func NewFileRegisty() *FileRegistry {
	return &FileRegistry{
		tree:      art.New(),
		minKeyLen: math.MaxInt,
	}
}

func (r *FileRegistry) Add(hdr FileHeader) {
	for _, sig := range hdr.Signatures {
		key := art.Key(sig)
		headers := r.get(key)

		r.tree.Insert(
			key,
			append(headers, hdr),
		)
		r.minKeyLen = min(r.minKeyLen, len(key))
	}
}

// TODO: consider wildcard matching for implementing offset

// Searches the registry for headers where the key matches a prefix of `data`.
// The search starts with `r.minKeyLen` and iteratively extends the key length
// as long as matching headers are found. Each found header is processed by `handleHeader`.
func (r *FileRegistry) Search(data []byte, handleHeader func(hdr FileHeader) uint64) uint64 {
	if r.tree.Size() == 0 {
		return 0
	}

	keyLen := min(r.minKeyLen, len(data))
	for {
		// Performance Note: The current search complexity can be higher than ideal.
		// For optimal performance, consider modifying the underlying tree implementation
		// to support an incremental search mechanism, which would reduce the complexity
		// to O(maxKeyLen) by avoiding repeated lookups for progressively longer keys.
		headers := r.get(data[:keyLen])
		for _, hdr := range headers {
			if size := handleHeader(hdr); size > 0 {
				return size
			}
		}

		if len(headers) == 0 {
			break
		}
		keyLen = min(keyLen+1, len(data))
	}
	return 0
}

func (r *FileRegistry) get(sig []byte) headers {
	value, found := r.tree.Search(art.Key(sig))
	if !found {
		return nil
	}
	return value.(headers)
}
