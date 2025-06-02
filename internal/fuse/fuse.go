//go:build linux
// +build linux

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
package fuse

import (
	"context"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

type FileEntry struct {
	Name   string
	Offset uint64
	Size   uint64
}

type RecoverFS struct {
	r io.ReaderAt

	mtx     sync.RWMutex
	entries map[string]FileEntry

	mountpoint string
}

func (fs *RecoverFS) Root() (fs.Node, error) {
	return &Dir{
		fs: fs,
	}, nil
}

// Dir implements both fs.Node and fs.HandleReadDirAller
type Dir struct {
	fs *RecoverFS
}

func (*Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0555
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if e, ok := d.fs.entries[name]; ok {
		return File{
			r:    io.NewSectionReader(d.fs.r, int64(e.Offset), int64(e.Size)),
			size: e.Size,
		}, nil
	}
	return nil, fuse.ENOENT
}

func (d Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.fs.mtx.RLock()
	defer d.fs.mtx.RUnlock()

	i := 0
	dirEntries := make([]fuse.Dirent, len(d.fs.entries))
	for _, e := range d.fs.entries {
		dirEntries[i] = fuse.Dirent{
			Inode: uint64(i),
			Name:  e.Name,
			Type:  fuse.DT_File,
		}
		i++
	}
	sort.Slice(dirEntries, func(i, j int) bool {
		return dirEntries[i].Name < dirEntries[j].Name
	})
	for i := range dirEntries {
		dirEntries[i].Inode = uint64(i)
	}
	return dirEntries, nil
}

// File implements both fs.Node and fs.HandleReader
type File struct {
	r    io.ReaderAt
	size uint64
}

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0444
	a.Size = f.size
	a.Mtime = time.Now()
	return nil
}

func (f File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	size := int(req.Size)
	offset := req.Offset

	if offset >= int64(f.size) {
		// Trying to read past EOF
		resp.Data = []byte{}
		return nil
	}

	// Clamp size if reading near EOF
	if offset+int64(size) > int64(f.size) {
		size = int(int64(f.size) - offset)
	}

	buf := make([]byte, size)

	n, err := f.r.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return err
	}

	resp.Data = buf[:n]
	return nil
}
