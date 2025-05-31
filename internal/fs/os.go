//go:build !windows
// +build !windows

package fs

import "os"

type OsFile struct {
	*os.File
}

func Open(path string) (File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &OsFile{File: f}, err
}
