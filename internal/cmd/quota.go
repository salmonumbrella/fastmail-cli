package cmd

import (
	"fmt"
	"strings"

	"github.com/salmonumbrella/fastmail-cli/internal/format"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newQuotaCmd(app *App) *cobra.Command {
	var formatFlag string

	cmd := &cobra.Command{
		Use:     "quota",
		Aliases: []string{"storage", "usage"},
		Short:   "Display storage quota and usage information",
		Long: `Display storage quota and usage information for your Fastmail account.

Shows used and available storage with a visual progress bar.

Examples:
  fastmail quota                    # Show quotas with human-readable sizes
  fastmail quota --format bytes     # Show raw byte values
  fastmail quota --format human     # Explicitly use human-readable format`,
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			quotas, err := client.GetQuotas(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get quotas: %w", err)
			}

			if len(quotas) == 0 {
				if app.IsJSON(cmd.Context()) {
					return app.PrintJSON(cmd, []any{})
				}
				printNoResults("No quota information available")
				return nil
			}

			// JSON output
			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, quotas)
			}

			// Human-readable output
			for i, quota := range quotas {
				if i > 0 {
					fmt.Println()
				}
				displayQuota(quota, formatFlag)
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&formatFlag, "format", "human", "Output format: human, bytes")

	return cmd
}

// displayQuota displays a single quota with formatting
func displayQuota(quota jmap.Quota, formatMode string) {
	tw := outfmt.NewTabWriter()

	// Display quota name/description
	title := quota.Name
	if quota.Description != "" {
		title = quota.Description
	}
	fmt.Printf("%s:\n", title)

	// Format values based on format flag
	var usedStr, limitStr, availStr string
	var percentage float64

	if formatMode == "bytes" {
		usedStr = fmt.Sprintf("%d bytes", quota.Used)
		if quota.Limit > 0 {
			limitStr = fmt.Sprintf("%d bytes", quota.Limit)
			availStr = fmt.Sprintf("%d bytes", quota.Limit-quota.Used)
			percentage = float64(quota.Used) / float64(quota.Limit) * 100
		} else {
			limitStr = "unlimited"
			availStr = "unlimited"
		}
	} else {
		// human format
		usedStr = format.FormatBytes(quota.Used)
		if quota.Limit > 0 {
			limitStr = format.FormatBytes(quota.Limit)
			availStr = format.FormatBytes(quota.Limit - quota.Used)
			percentage = float64(quota.Used) / float64(quota.Limit) * 100
		} else {
			limitStr = "unlimited"
			availStr = "unlimited"
		}
	}

	// Display usage information
	if quota.Limit > 0 {
		fmt.Fprintf(tw, "  Used:\t%s / %s (%.1f%%)\n", usedStr, limitStr, percentage)
		fmt.Fprintf(tw, "  Available:\t%s\n", availStr)
	} else {
		fmt.Fprintf(tw, "  Used:\t%s\n", usedStr)
		fmt.Fprintf(tw, "  Limit:\t%s\n", limitStr)
	}
	tw.Flush()

	// Display progress bar
	if quota.Limit > 0 {
		fmt.Println()
		fmt.Printf("  %s %.1f%%\n", progressBar(percentage), percentage)
	}
}

// progressBar creates a visual progress bar
func progressBar(percentage float64) string {
	const barWidth = 40
	filled := int(percentage / 100 * barWidth)

	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled)
	return fmt.Sprintf("[%s]", bar)
}
