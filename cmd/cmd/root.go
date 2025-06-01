package cmd

import (
	"github.com/spf13/cobra"
)

const AppName = "diglet"

func Execute() error {
	rootCmd := &cobra.Command{
		Use: AppName,
	}

	rootCmd.AddCommand(DefineScanCommand())
	rootCmd.AddCommand(DefineMountCommand())

	return rootCmd.Execute()
}
