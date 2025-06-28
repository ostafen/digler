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
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"os"

	"github.com/ostafen/digler/internal/logger"
	osutils "github.com/ostafen/digler/pkg/util/os"
	"github.com/spf13/cobra"
)

func DefineMergeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge <file1> <file2> ...",
		Short: "Merge multiple files into a single disk image",
		Long: `The 'merge' command combines multiple files into a single flat disk image.
This is useful for testing scanners and plugins with known, reproducible data.
By default, files are concatenated in the order given. You can optionally add zero-byte padding between files to simulate gaps.`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE:         RunMerge,
	}

	cmd.Flags().StringP("output", "o", "", "Path to the output disk image file (required)")
	cmd.Flags().Int64("padding", 0, "Number of zero bytes to insert between files (optional)")
	cmd.Flags().Int("min-gap", 4*1024, "minimum gap size in bytes between files")
	cmd.Flags().Int("max-gap", 512*1024, "maximum gap size in bytes between files")
	cmd.Flags().Int("block-size", 512, "block size in bytes")

	_ = cmd.MarkFlagRequired("output")

	return cmd
}

func RunMerge(cmd *cobra.Command, args []string) error {
	filePaths := make([]string, 0, len(args))
	for _, arg := range args {
		paths, err := osutils.ListFiles(arg)
		if err != nil {
			return err
		}
		filePaths = append(filePaths, paths...)
	}

	out, _ := cmd.Flags().GetString("output")

	minGap, _ := cmd.Flags().GetInt("min-gap")
	maxGap, _ := cmd.Flags().GetInt("max-gap")

	if minGap > maxGap {
		return fmt.Errorf("min-gap (%d) cannot be greater than max-gap (%d)", minGap, maxGap)
	}
	if minGap <= 0 {
		return fmt.Errorf("min-gap must be greater than 0")
	}

	blockSize, _ := cmd.Flags().GetInt("block-size")
	if blockSize <= 0 {
		return fmt.Errorf("block size must be greater than 0")
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	logger := logger.New(os.Stdout, logger.InfoLevel)

	logger.Infof("Merging %d files into %s", len(filePaths), out)

	w := bufio.NewWriter(f)

	gapSize := minGap + mrand.IntN(maxGap-minGap+1)
	// Ensure gap size is a multiple of block size
	gapSize = min(1, gapSize/blockSize) * blockSize

	bytesWritten := int64(0)
	for _, path := range filePaths {
		_, err := io.CopyN(w, rand.Reader, int64(gapSize))
		if err != nil {
			return err
		}
		bytesWritten += int64(gapSize)

		nCopied, err := osutils.CopyFile(w, path)
		if err != nil {
			return err
		}
		bytesWritten += nCopied

		padding := int64(blockSize) - nCopied%int64(blockSize)

		gapSize = minGap + mrand.IntN(maxGap-minGap+1)
		// Ensure next file starts at a block boundary
		gapSize = min(1, gapSize/blockSize) * blockSize
		gapSize += int(padding)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("error flushing writer: %w", err)
	}

	logger.Infof("Merging successfully completed. %d bytes written.", bytesWritten)
	return nil
}
