package cmd

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  `Display version information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Version info is already printed by PersistentPreRunE
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
