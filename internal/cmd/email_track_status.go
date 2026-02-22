package cmd

import (
	"fmt"
	"strings"

	"github.com/salmonumbrella/fastmail-cli/internal/tracking"
	"github.com/spf13/cobra"
)

func newEmailTrackStatusCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show tracking configuration status",
		RunE: runE(app, func(cmd *cobra.Command, _ []string, app *App) error {
			cfg, err := tracking.LoadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			path, _ := tracking.ConfigPath()

			if !cfg.IsConfigured() {
				if app.IsJSON(cmd.Context()) {
					return app.PrintJSON(cmd, map[string]any{
						"configured": false,
						"configPath": path,
					})
				}

				if path != "" {
					fmt.Printf("config_path\t%s\n", path)
				}
				fmt.Printf("configured\tfalse\n")
				return nil
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"configured":      true,
					"configPath":      path,
					"workerUrl":       cfg.WorkerURL,
					"adminConfigured": strings.TrimSpace(cfg.AdminKey) != "",
				})
			}

			if path != "" {
				fmt.Printf("config_path\t%s\n", path)
			}
			fmt.Printf("configured\ttrue\n")
			fmt.Printf("worker_url\t%s\n", cfg.WorkerURL)
			fmt.Printf("admin_configured\t%t\n", strings.TrimSpace(cfg.AdminKey) != "")

			return nil
		}),
	}

	return cmd
}
