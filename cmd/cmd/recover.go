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
	"os"
	"path/filepath"
	"strings"

	"github.com/ostafen/digler/internal/fs"
	"github.com/ostafen/digler/internal/logger"
	"github.com/ostafen/digler/internal/scan"
	"github.com/ostafen/digler/pkg/dfxml"
	osutils "github.com/ostafen/digler/pkg/util/os"
	"github.com/spf13/cobra"
)

func DefineRecoverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recover <image_path> <report_file>",
		Short: "Recover files from a disk image using a scan report",
		Long: `The 'recover' command extracts files from a disk image or device based on the information provided in a scan report.
The scan report contains metadata and file information needed for recovery.
You must provide the full path to the image file and the report file.
Recovered files will be saved to the specified output directory.`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE:         RunRecover,
	}
	cmd.Flags().StringP("output-dir", "i", "", "Absolute path to the directory where recovered data will be placed.")
	return cmd
}

func RunRecover(cmd *cobra.Command, args []string) error {
	f, err := fs.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	reportFile, err := os.Open(args[1])
	if err != nil {
		return err
	}

	objects, err := dfxml.ReadFileObjects(bufio.NewReader(reportFile))
	if err != nil {
		return err
	}

	outDir, _ := cmd.Flags().GetString("output-dir")
	if outDir == "" {
		wdir, err := os.Getwd()
		if err != nil {
			return err
		}

		base := filepath.Base(reportFile.Name())
		name := strings.TrimSuffix(base, filepath.Ext(base))
		outDir = filepath.Join(wdir, name+"-dump")
	}

	_, err = osutils.EnsureDir(outDir, true)
	if err != nil {
		return err
	}

	finfos, err := fileObjectsToFileInfo(objects)
	if err != nil {
		return err
	}

	logger := logger.New(os.Stdout, logger.InfoLevel)

	for _, finfo := range finfos {
		logger.Infof("recovering file %s", filepath.Join(outDir, finfo.Name))

		if err := scan.DumpFile(f, outDir, &finfo); err != nil {
			logger.Errorf("unable to dump file %s: %s", finfo.Name, err)
		}
	}
	return nil
}
