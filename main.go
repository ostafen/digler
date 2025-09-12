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
package main

import (
	"embed"

	"github.com/ostafen/digler/internal/app"
	"github.com/ostafen/digler/internal/app/api"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	AppName = "Digler"

	Width  = 900
	Height = 900
)

func main() {
	sysAPI := &api.SystemAPI{}
	scanAPI := &api.ScanAPI{}

	app := &app.App{}

	err := wails.Run(&options.App{
		Title:         AppName,
		Width:         Width,
		Height:        Height,
		DisableResize: true,
		Assets:        assets,
		OnStartup:     app.Startup,
		OnShutdown:    app.Shutdown,
		Bind: []any{
			app,
			sysAPI,
			scanAPI,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
		},
	})
	exitOnError(err)
}

func exitOnError(err error) {
	if err != nil {
		panic(err)
	}
}
