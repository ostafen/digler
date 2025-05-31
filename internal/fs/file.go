package fs

import (
	"io"
	"os"
)

type File interface {
	io.ReadCloser
	io.ReaderAt
	Stat() (os.FileInfo, error)
}
