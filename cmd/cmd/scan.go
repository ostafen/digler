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
	"log/slog"
	"math"

	"github.com/ostafen/digler/internal/disk"
	"github.com/ostafen/digler/internal/scan"
	"github.com/spf13/cobra"
)

func DefineScanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "scan <device>",
		Short:        "List files in a vault layer",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE:         RunScan,
	}

	cmd.Flags().StringP("dump", "d", "", "dump the found files to the specified directory")
	cmd.Flags().Uint64("block-size", 0, "enforce a specific block size during scanning")
	cmd.Flags().Uint64("scan-buffer-size", 4*1024*1024, "the size of the scan buffer")
	cmd.Flags().Uint64("max-scan-size", math.MaxUint64, "max number of bytes to scan")
	cmd.Flags().Bool("no-log", false, "disable logging")
	cmd.Flags().StringP("output", "o", "", "The path of the scan index file")
	return cmd
}

func RunScan(cmd *cobra.Command, args []string) error {
	path := disk.NormalizeVolumePath(args[0])
	opts := parseOptions(cmd)

	return scan.Scan(path, opts)
}

func parseOptions(cmd *cobra.Command) scan.Options {
	dumpDir := cmd.Flag("dump").Value.String()
	disableLog, _ := cmd.Flags().GetBool("no-log")
	outputFile, _ := cmd.Flags().GetString("output")
	scanBufferSize, _ := cmd.Flags().GetUint64("scan-buffer-size")
	blockSize, _ := cmd.Flags().GetUint64("block-size")
	maxScanSize, _ := cmd.Flags().GetUint64("max-scan-size")
	logLevel, _ := cmd.Flags().GetString("log-level")

	return scan.Options{
		DumpDir:        dumpDir,
		ReportFile:     outputFile,
		BlockSize:      blockSize,
		MaxScanSize:    maxScanSize,
		ScanBufferSize: scanBufferSize,
		DisableLog:     disableLog,
		LogLevel:       slogLevel(logLevel),
	}
}

func slogLevel(level string) slog.Level {
	switch level {
	case "INFO":
		return slog.LevelInfo
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	}
	return slog.LevelInfo
}
