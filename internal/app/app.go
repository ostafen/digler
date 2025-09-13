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

	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	runtimeFilters := make([]runtime.FileFilter, len(filters))
	for i, f := range filters {
		runtimeFilters[i] = runtime.FileFilter{DisplayName: f.Name, Pattern: f.Pattern}
	}

	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: runtimeFilters,
	})
}

// OpenFolderDialog opens a folder picker
func (a *App) OpenFolderDialog() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select output folder",
	})
}
