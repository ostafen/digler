//go:build !windows
// +build !windows

package fs

import "os"

func Open(path string) (File, error) {
	return os.Open(path)
}
