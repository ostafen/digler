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
