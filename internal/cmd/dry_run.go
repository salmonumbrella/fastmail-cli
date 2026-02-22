package cmd

import "github.com/spf13/cobra"

func printDryRunList(app *App, cmd *cobra.Command, header, key string, items []string, extra map[string]any) error {
	if app.IsJSON(cmd.Context()) {
		payload := map[string]any{
			"dryRun": true,
			key:      items,
		}
		for k, v := range extra {
			payload[k] = v
		}
		return app.PrintJSON(cmd, payload)
	}

	printList(header, items)
	return nil
}
