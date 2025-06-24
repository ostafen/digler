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
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"github.com/ostafen/digler/internal/format"
	osutils "github.com/ostafen/digler/pkg/util/os"
)

func Mount(mountpoint string, r io.ReaderAt, finfos []format.FileInfo) error {
	created, err := osutils.EnsureDir(mountpoint, true)
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
