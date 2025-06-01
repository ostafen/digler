//go:build !linux
// +build !linux

package fuse

import (
	"fmt"
	"io"

	"github.com/ostafen/digler/internal/format"
)

func Mount(mountpoint string, r io.ReaderAt, entries []format.FileInfo) error {
	return fmt.Errorf("FUSE mount is only supported on Linux")
}
