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
	"context"
	"embed"
	"path/filepath"
	"strings"

	"github.com/ostafen/digler/internal/app"
	"github.com/ostafen/digler/internal/app/api"
	"github.com/ostafen/digler/internal/app/store"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	AppName    = "Digler"
	HistoryDir = "scan_history"
	ConfigFile = "config.json"

	Width  = 900
	Height = 900
)

func main() {
	homeDir, err := app.UserHomeDir()
	exitOnError(err)

	dataDir, user, err := app.UserDataDir(AppName)
	exitOnError(err)

	scanHistoryStore, err := store.NewStore[api.ScanRecord](
		filepath.Join(dataDir, "scan_history"),
		user,
		100,
	)
	exitOnError(err)

	configStore, err := store.NewConfigStore(filepath.Join(dataDir, ConfigFile))
	exitOnError(err)

	sysAPI := api.NewSystemAPI(dataDir, filepath.Join(homeDir, strings.ToLower(AppName)))
	configAPI := api.NewConfigAPI(configStore)
	scanAPI := api.NewScanAPI(scanHistoryStore)

	app := &app.App{}

	err = wails.Run(&options.App{
		Title:         AppName,
		Width:         Width,
		Height:        Height,
		DisableResize: true,
		Assets:        assets,
		OnStartup:     app.Startup,
		OnShutdown: func(ctx context.Context) {
			configStore.Close()
			scanHistoryStore.Close()
		},
		Bind: []any{
			app,
			sysAPI,
			configAPI,
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
