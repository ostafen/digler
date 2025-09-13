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
package os

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EnsureDir checks if the specified directory exists, and optionally verifies if it is empty.
// If the directory does not exist, it attempts to create it with 0755 permissions.
func EnsureDir(dir string, empty bool) (bool, error) {
	finfo, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			return false, fmt.Errorf("failed to create mountpoint %s: %w", dir, err)
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to stat mountpoint %s: %w", dir, err)
	}

	if !finfo.IsDir() {
		return false, fmt.Errorf("mountpoint %s is not a directory", dir)
	}

	if !empty {
		return false, nil
	}

	isEmpty, err := IsDirEmpty(dir)
	if err != nil {
		return false, fmt.Errorf("failed to check if directory %s is empty: %w", dir, err)
	}

	if !isEmpty {
		return false, fmt.Errorf("directory %s is not empty", dir)
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

// ListFiles takes a path and returns a slice of file paths.
// If the path is a regular file, it returns []string{path}.
// If it's a directory, it returns all regular files in that directory (non-recursive).
func ListFiles(path string) ([]string, error) {
	finfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %s: %w", path, err)
	}

	if finfo.Mode().IsRegular() {
		return []string{path}, nil
	}

	if !finfo.IsDir() {
		return nil, fmt.Errorf("path %s is neither a regular file nor a directory", path)
	}

	files := []string{}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		filePath := filepath.Join(path, entry.Name())
		files = append(files, filePath)
	}
	return files, nil
}

func CopyFile(dst io.Writer, filePath string) (int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return -1, err
	}
	defer f.Close()

	return io.Copy(dst, f)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func AtomicWriteFile(filePath string, writeFile func(f *os.File) error) error {
	tempFile, err := os.CreateTemp("", "tmp-*.json")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if err := writeFile(tempFile); err != nil {
		tempFile.Close()
		return err
	}
	_ = tempFile.Close()

	return os.Rename(tempPath, filePath)
}
