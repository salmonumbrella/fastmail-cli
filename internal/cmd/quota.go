package cmd

import (
	"fmt"
	"math"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/spf13/cobra"
)

func newQuotaCmd(flags *rootFlags) *cobra.Command {
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
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			quotas, err := client.GetQuotas(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get quotas: %w", err)
			}

			if len(quotas) == 0 {
				if isJSON(cmd.Context()) {
					return printJSON(cmd, []any{})
				}
				fmt.Println("No quota information available")
				return nil
			}

			// JSON output
			if isJSON(cmd.Context()) {
				return printJSON(cmd, quotas)
			}

			// Human-readable output
			for i, quota := range quotas {
				if i > 0 {
					fmt.Println()
				}
				displayQuota(quota, formatFlag)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&formatFlag, "format", "human", "Output format: human, bytes")

	return cmd
}

// displayQuota displays a single quota with formatting
func displayQuota(quota jmap.Quota, format string) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	// Display quota name/description
	title := quota.Name
	if quota.Description != "" {
		title = quota.Description
	}
	fmt.Printf("%s:\n", title)

	// Format values based on format flag
	var usedStr, limitStr, availStr string
	var percentage float64

	if format == "bytes" {
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
		usedStr = formatBytes(quota.Used)
		if quota.Limit > 0 {
			limitStr = formatBytes(quota.Limit)
			availStr = formatBytes(quota.Limit - quota.Used)
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

// formatBytes converts bytes to human-readable format (KB, MB, GB, TB)
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	// Calculate the appropriate unit
	exp := int(math.Log(float64(bytes)) / math.Log(unit))

	// Ensure we don't go beyond TB
	if exp > 4 {
		exp = 4
	}

	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes) / math.Pow(unit, float64(exp))

	// Format with 1 decimal place
	return fmt.Sprintf("%.1f %s", value, units[exp])
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
