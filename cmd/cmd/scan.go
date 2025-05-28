package cmd

import (
	"github.com/ostafen/diglet/internal/scan"
	"github.com/spf13/cobra"
)

func DefineScanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "scan <device>",
		Short:         "List files in a vault layer",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		RunE:          RunScan,
	}

	cmd.Flags().StringP("dump", "d", "", "Dump the found files to the specified directory")
	return cmd
}

func RunScan(cmd *cobra.Command, args []string) error {
	path := args[0]

	dumpDir := cmd.Flag("dump").Value.String()
	dumpDirChanged := cmd.Flags().Changed("dump")

	if dumpDirChanged && dumpDir == "" {
		dumpDir = "dump"
	}
	return scan.Scan(path, dumpDir)
}
