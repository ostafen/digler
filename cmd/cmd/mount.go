package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ostafen/digler/internal/format"
	"github.com/ostafen/digler/internal/fs"
	"github.com/ostafen/digler/internal/fuse"
	"github.com/ostafen/digler/pkg/dfxml"
	"github.com/spf13/cobra"
)

func DefineMountCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mount <image_path> <report_file>",
		Short: "Mount a disk image to a specified mountpoint",
		Long: `The 'mount' command mounts a disk image or device based on the information provided in a report file.
The report file typically contains details about the image's structure and any required offsets.
You must provide the full path to the image file and the report file.`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE:         RunMount,
	}

	cmd.Flags().StringP("mountpoint", "m", "", "Absolute path to the directory where the filesystem will be mounted. If not specified, a default will be generated.")
	return cmd
}

func RunMount(cmd *cobra.Command, args []string) error {
	f, err := fs.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	reportFile, err := os.Open(args[1])
	if err != nil {
		return err
	}

	mountpoint, _ := cmd.Flags().GetString("mountpoint")
	if mountpoint == "" {
		mountpoint = getMountpoint(reportFile.Name())
	}

	objects, err := dfxml.ReadFileObjects(bufio.NewReader(reportFile))
	if err != nil {
		return err
	}

	finfos, err := fileObjectsToFileInfo(objects)
	if err != nil {
		return err
	}
	return fuse.Mount(mountpoint, f, finfos)
}

// getMountpoint generates a mountpoint name from a report file name by stripping the extension.
// If the extension is empty, "_mnt" is added.
func getMountpoint(reportFileName string) string {
	baseName := filepath.Base(reportFileName)
	ext := filepath.Ext(baseName)
	baseName = strings.TrimSuffix(baseName, ext)
	mountpoint := baseName
	if ext == "" {
		mountpoint += "_mnt"
	}
	return mountpoint
}

func fileObjectsToFileInfo(objs []dfxml.FileObject) ([]format.FileInfo, error) {
	finfos := make([]format.FileInfo, len(objs))
	for i, o := range objs {
		runs := o.ByteRuns.Runs
		if len(runs) < 1 {
			return nil, fmt.Errorf("invalid report file")
		}

		finfos[i] = format.FileInfo{
			Name:   o.Filename,
			Offset: runs[0].Offset,
			Size:   runs[0].Length,
		}
	}
	return finfos, nil
}
