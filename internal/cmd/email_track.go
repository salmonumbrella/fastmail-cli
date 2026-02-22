package cmd

import (
	"github.com/spf13/cobra"
)

func newEmailTrackCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "track",
		Short: "Email open tracking",
		Long:  `Commands for setting up and querying email open tracking.`,
	}

	cmd.AddCommand(newEmailTrackSetupCmd(app))
	cmd.AddCommand(newEmailTrackOpensCmd(app))
	cmd.AddCommand(newEmailTrackRotateCmd(app))
	cmd.AddCommand(newEmailTrackStatusCmd(app))

	return cmd
}
