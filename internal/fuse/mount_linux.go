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
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"github.com/ostafen/digler/internal/format"
)

func Mount(mountpoint string, r io.ReaderAt, finfos []format.FileInfo) error {
	created, err := PrepareMountpoint(mountpoint)
	if err != nil {
		return err
	}
	if created {
		defer os.Remove(mountpoint)
	}

	c, err := fuse.Mount(mountpoint)
	if err != nil {
		return err
	}
	defer c.Close()

	entries := make(map[string]FileEntry, len(finfos))
	for _, e := range finfos {
		entries[e.Name] = FileEntry{
			Name:   e.Name,
			Offset: e.Offset,
			Size:   e.Size,
		}
	}

	fs := &RecoverFS{
		r:          r,
		entries:    entries,
		mountpoint: mountpoint,
	}

	go func() {
		srv := fusefs.New(c, nil)
		if err := srv.Serve(fs); err != nil {
			log.Fatalf("Serve error: %v", err)
		}
	}()
	return waitForUmount(mountpoint)
}

func waitForUmount(mountpoint string) error {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)

	log.Println("Waiting for termination signal...")

	const maxUnmountRetries = 3

	unmountAttempts := 0
	for sig := range sigc {
		log.Printf("Signal received: %v.", sig)

		if unmountAttempts >= maxUnmountRetries-1 {
			log.Fatalf("Maximum unmount retries (%d) exceeded. Still unable to unmount %s. Forcefully exiting.",
				maxUnmountRetries, mountpoint)
		}

		log.Printf("Attempting unmount of %s (attempt %d/%d)...", mountpoint, unmountAttempts+1, maxUnmountRetries)
		err := fuse.Unmount(mountpoint)
		if err == nil {
			log.Println("Unmounted successfully, exiting.")
			return nil
		}

		unmountAttempts++
		log.Printf("Unmount failed: %v. Remaining retries: %d. Waiting for another signal to retry...", err, maxUnmountRetries-unmountAttempts)
	}
	return nil
}

// PrepareMountpoint ensures the given path is a valid, empty directory suitable for FUSE mounting.
// It creates the directory if it doesn't exist. Returns `true` if created, `false` otherwise,
// or an error if the path exists but isn't an empty directory.
func PrepareMountpoint(mountpoint string) (bool, error) {
	finfo, err := os.Stat(mountpoint)
	if errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(mountpoint, 0755)
		if err != nil {
			return false, fmt.Errorf("failed to create mountpoint %s: %w", mountpoint, err)
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to stat mountpoint %s: %w", mountpoint, err)
	}

	if !finfo.IsDir() {
		return false, fmt.Errorf("mountpoint %s is not a directory", mountpoint)
	}

	empty, err := IsDirEmpty(mountpoint)
	if err != nil {
		return false, fmt.Errorf("failed to check if mountpoint %s is empty: %w", mountpoint, err)
	}

	if !empty {
		return false, fmt.Errorf("mountpoint %s is not empty", mountpoint)
	}
	return false, nil
}

// IsDirEmpty returns true if the directory at path is empty, false otherwise.
// Returns an error if the path does not exist or is not a directory.
func IsDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	entries, err := f.Readdir(1)
	if err != nil {
		if err == io.EOF {
			return true, nil
		}
		return false, err
	}

	if len(entries) > 0 {
		return false, nil
	}
	return true, nil
}
