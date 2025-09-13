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
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	osutils "github.com/ostafen/digler/pkg/util/os"
)

var ErrKeyNotFound = errors.New("key not found")

// Store manages an history store with one file per record and keeps directory index in memory as a ring buffer.
type Store[T any] struct {
	dir        string
	maxRecords int
	fileDir    map[uint64]struct{}
	buf        *RingBuffer[uint64] // sorted in ascending order
	user       *user.User
}

// NewStore initializes the store directory and loads existing entries.
func NewStore[T any](appName string, maxRecords int) (*Store[T], error) {
	baseDir, user, err := userHistoryDir(appName)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}

	s := &Store[T]{
		dir:        baseDir,
		maxRecords: maxRecords,
		fileDir:    make(map[uint64]struct{}, maxRecords),
		buf:        NewRingBuffer[uint64](maxRecords),
		user:       user,
	}

	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store[T]) init() error {
	// load existing files in directory
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	timestamps := make([]uint64, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ts, err := strconv.ParseUint(e.Name()[:len(e.Name())-5], 10, 64) // strip .json
		if err != nil {
			continue
		}
		timestamps = append(timestamps, ts)
		s.fileDir[ts] = struct{}{}
	}

	// keep only last maxRecords, delete older files
	if len(timestamps) > s.maxRecords {
		extra := len(timestamps) - s.maxRecords
		for i := range extra {
			ts := timestamps[i]

			if err := os.Remove(s.fileName(ts)); err != nil {
				log.Println("unable to remove file at path: " + err.Error())
			}
			delete(s.fileDir, ts)
		}
		timestamps = timestamps[extra:]
	}

	for _, t := range timestamps {
		s.buf.Append(t)
	}
	return nil
}

// Append adds a new scan record safely using a temp file created by os.CreateTemp, overwriting oldest if buffer is full.
func (s *Store[T]) Append(record T, ts uint64) error {
	filePath := filepath.Join(s.dir, strconv.FormatUint(ts, 10)+".json")

	err := osutils.AtomicWriteFile(filePath, func(f *os.File) error {
		enc := json.NewEncoder(f)
		return enc.Encode(record)
	})
	if err != nil {
		return err
	}

	if err := s.chown(filePath); err != nil {
		return err
	}

	oldTs, replaced := s.buf.Append(ts)
	if replaced {
		delete(s.fileDir, oldTs)
	}
	s.fileDir[ts] = struct{}{}
	return nil
}

func (s *Store[T]) chown(filePath string) error {
	if s.user == nil {
		return nil
	}

	uid, _ := strconv.Atoi(s.user.Uid)
	gid, _ := strconv.Atoi(s.user.Gid)

	return os.Chown(filePath, uid, gid)
}

func (s *Store[T]) Get(ts uint64) (T, error) {
	_, has := s.fileDir[ts]
	if !has {
		var zero T
		return zero, ErrKeyNotFound
	}
	return s.get(ts)
}

func (s *Store[T]) get(ts uint64) (T, error) {
	var rec T

	data, err := os.ReadFile(s.fileName(ts))
	if err != nil {
		return rec, err
	}
	err = json.Unmarshal(data, &rec)
	return rec, err
}

// LoadLast loads the last N scan records in descending order (most recent first).
func (s *Store[T]) LoadLast(n int) ([]T, error) {
	if n > s.buf.Len() {
		n = s.buf.Len()
	}

	records := make([]T, n)
	for i := 0; i < n; i++ {
		ts := s.buf.Get(s.buf.Len() - i - 1)

		rec, err := s.get(ts)
		if err != nil {
			return nil, err
		}
		records[i] = rec
	}
	return records, nil
}

func (s *Store[T]) RemoveAll() error {
	for ts := range s.fileDir {
		os.Remove(s.fileName(ts))

		delete(s.fileDir, ts)
	}

	s.buf.Reset()
	return nil
}

func (s *Store[T]) Close() error {
	return nil
}

func (s *Store[T]) fileName(ts uint64) string {
	return filepath.Join(s.dir, strconv.FormatUint(ts, 10)+".json")
}

// userHistoryDir returns a directory path to store user-specific history.
// Works on Linux, macOS, Windows, with or without sudo/admin.
func userHistoryDir(appName string) (string, *user.User, error) {
	var u *user.User
	var home string

	switch runtime.GOOS {
	case "windows":
		home = os.Getenv("LOCALAPPDATA")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		if home == "" {
			return "", nil, fmt.Errorf("unable to determine home directory on Windows")
		}
	default:
		var err error
		sudoUser := os.Getenv("SUDO_USER")
		if sudoUser != "" {
			u, err = user.Lookup(sudoUser)
			if err != nil {
				return "", nil, fmt.Errorf("failed to lookup sudo user %s: %w", sudoUser, err)
			}
			home = u.HomeDir
		} else {
			home, err = os.UserHomeDir()
			if err != nil {
				return "", nil, fmt.Errorf("failed to get current user home directory: %w", err)
			}
		}
	}

	historyDir := filepath.Join(home, fmt.Sprintf(".%s_history", strings.ToLower(appName)))
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		return "", u, fmt.Errorf("failed to create history directory %s: %w", historyDir, err)
	}
	return historyDir, u, nil
}
