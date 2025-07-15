package cmd

import (
	"fmt"

	"github.com/bilbo290/automagic/pkg/daemon"
	"github.com/bilbo290/automagic/pkg/enum"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Daemon mode commands",
	Long:  `Commands for running automagic in daemon mode`,
}

var startDaemonCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon mode",
	Long:  `Start daemon mode to monitor for issues with 'claude' label`,
	RunE: func(cmd *cobra.Command, args []string) error {
		runTypeFlag, _ := cmd.Flags().GetString("run-type")

		runType, valid := enum.ParseRunType(runTypeFlag)
		if !valid {
			return fmt.Errorf("invalid run type '%s'", runTypeFlag)
		}

		var d *daemon.Daemon
		switch runType {
		case enum.RunTypeDryRun:
			d = daemon.New(providerInstance, cfg, enum.RunTypeDryRun)
		case enum.RunTypeSemiDryRun:
			d = daemon.New(providerInstance, cfg, enum.RunTypeSemiDryRun)
		default:
			d = daemon.New(providerInstance, cfg, enum.RunTypeNormal)
		}

		return d.Run()
	},
}

func init() {
	// Add flags
	startDaemonCmd.Flags().String("run-type", "normal", "Execution mode: normal, dry-run, semi-dry-run")

	// Add subcommands
	daemonCmd.AddCommand(startDaemonCmd)

	rootCmd.AddCommand(daemonCmd)
}
