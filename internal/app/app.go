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
package app

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

type FileDialogFilter struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
}

// OpenFileDialog opens a native file picker and returns the absolute path
func (a *App) OpenFileDialog(title string, filters []FileDialogFilter) (string, error) {
	runtimeFilters := make([]wailsruntime.FileFilter, len(filters))
	for i, f := range filters {
		runtimeFilters[i] = wailsruntime.FileFilter{DisplayName: f.Name, Pattern: f.Pattern}
	}

	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   title,
		Filters: runtimeFilters,
	})
}

// OpenFolderDialog opens a folder picker
func (a *App) OpenFolderDialog() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select output folder",
	})
}

// UserDataDir returns a directory path to store user-specific history.
// Works on Linux, macOS, Windows, with or without sudo/admin.
func UserDataDir(appName string) (string, *user.User, error) {
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

	dataDir := strings.ToLower(appName) + "_data"
	if runtime.GOOS != "windows" {
		dataDir = "." + dataDir
	}

	historyDir := filepath.Join(home, dataDir)
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		return "", u, fmt.Errorf("failed to create history directory %s: %w", historyDir, err)
	}
	return historyDir, u, nil
}

func UserHomeDir() (string, error) {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err != nil {
			return "", fmt.Errorf("failed to lookup sudo user %s: %w", sudoUser, err)
		}
		return u.HomeDir, nil
	}
	return os.UserHomeDir()
}
