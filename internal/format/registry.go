package format

import (
	"github.com/ostafen/digler/pkg/table"
)

type FileRegistry struct {
	table *table.PrefixTable[headers]
}

type headers []FileHeader

func NewFileRegisty() *FileRegistry {
	return &FileRegistry{
		table: table.New[headers](),
	}
}

func (r *FileRegistry) Add(hdr FileHeader) {
	for _, sig := range hdr.Signatures {
		headers, _ := r.table.Get(sig)

		r.table.Insert(
			sig,
			append(headers, hdr),
		)
	}
}

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
