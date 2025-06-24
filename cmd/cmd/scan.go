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
package cmd

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/ostafen/digler/internal/disk"
	"github.com/ostafen/digler/internal/logger"
	"github.com/ostafen/digler/internal/scan"
	"github.com/ostafen/digler/pkg/util/format"
	"github.com/spf13/cobra"
)

func DefineScanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "scan <device>",
		Short:        "Scan an image file or disk",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE:         RunScan,
	}

	cmd.Flags().StringP("dump", "d", "", "dump the found files to the specified directory")
	cmd.Flags().String("block-size", "0", "use the specified block size during scanning")
	cmd.Flags().String("scan-buffer-size", "4MB", "the size of the scan buffer")
	cmd.Flags().String("max-scan-size", "", "max number of bytes to scan")
	cmd.Flags().String("max-file-size", "4GB", "maximum size of a carved file")
	cmd.Flags().Bool("no-log", false, "disable logging")
	cmd.Flags().StringSliceP("ext", "", nil, "file extensions to parse")
	cmd.Flags().StringP("output", "o", "", "The path of the scan index file")
	cmd.Flags().StringSlice("plugins", nil, "paths to plugin .so files or directories containing plugins")

	return cmd
}

func RunScan(cmd *cobra.Command, args []string) error {
	path := disk.NormalizeVolumePath(args[0])

	opts, err := parseOptions(cmd)
	if err != nil {
		return err
	}
	return scan.Scan(path, opts)
}

func parseOptions(cmd *cobra.Command) (scan.Options, error) {
	dumpDir := cmd.Flag("dump").Value.String()
	disableLog, _ := cmd.Flags().GetBool("no-log")
	outputFile, _ := cmd.Flags().GetString("output")

	scanBufferSize := getBytes(cmd, "scan-buffer-size")
	blockSize := getBytes(cmd, "block-size")
	maxScanSize := getBytes(cmd, "max-scan-size")
	maxFileSize := getBytes(cmd, "max-file-size")

	fileExt, _ := cmd.Flags().GetStringSlice("ext")
	logLevel, _ := cmd.Flags().GetString("log-level")

	plugins, _ := cmd.Flags().GetStringSlice("plugins")

	pluginPaths, err := listPlugins(plugins)
	if err != nil {
		return scan.Options{}, nil
	}

	return scan.Options{
		DumpDir:        dumpDir,
		ReportFile:     outputFile,
		BlockSize:      blockSize,
		MaxScanSize:    maxScanSize,
		ScanBufferSize: scanBufferSize,
		MaxFileSize:    maxFileSize,
		DisableLog:     disableLog,
		FileExt:        fileExt,
		Plugins:        pluginPaths,
		LogLevel:       logger.ParseLevel(logLevel),
	}, nil
}

func getBytes(cmd *cobra.Command, name string) uint64 {
	s, _ := cmd.Flags().GetString(name)

	v, err := format.ParseBytes(s)
	if err != nil {
		return math.MaxUint64
	}
	return v
}

// listPlugins expands plugin paths: if path is a file, add it directly;
// if path is a directory, scan it recursively for .so files.
func listPlugins(plugins []string) ([]string, error) {
	var pluginPaths []string

	for _, p := range plugins {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			if !strings.HasSuffix(info.Name(), ".so") {
				return nil, fmt.Errorf("plugin file %s does not have .so extension", info.Name())
			}
			pluginPaths = append(pluginPaths, p)
			continue
		}

		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".so") {
				pluginPaths = append(pluginPaths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return pluginPaths, nil
}
