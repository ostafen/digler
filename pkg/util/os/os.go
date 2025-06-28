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
