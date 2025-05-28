package cmd

import (
	"github.com/spf13/cobra"
)

const AppName = "diglet"

func Execute() error {
	rootCmd := &cobra.Command{
		Use:   AppName,
		Short: AppName + " - disk analysis and recovery tool",
	}

	rootCmd.AddCommand(DefineScanCommand())

	return rootCmd.Execute()
}
