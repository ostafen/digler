package format

import (
	"math"

	"github.com/ostafen/digler/pkg/table"
	art "github.com/plar/go-adaptive-radix-tree/v2"
)

type FileRegistry struct {
	table     *table.PrefixTable[headers]
	minKeyLen int
}

type headers []FileHeader

func NewFileRegisty() *FileRegistry {
	return &FileRegistry{
		table:     table.New[headers](),
		minKeyLen: math.MaxInt,
	}
}

func (r *FileRegistry) Add(hdr FileHeader) {
	for _, sig := range hdr.Signatures {
		key := art.Key(sig)
		headers, _ := r.table.Get(sig)

		r.table.Insert(
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
func (r *FileRegistry) Search(data []byte, handleHeader func(hdr FileHeader) bool) {
	if r.table.Size() == 0 {
		return
	}

	r.table.Walk(data, func(hdrs headers) bool {
		for _, hdr := range hdrs {
			if handleHeader(hdr) {
				return true
			}
		}
		return false
	})
}
