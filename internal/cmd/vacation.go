package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/spf13/cobra"
)

func newVacationCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vacation",
		Aliases: []string{"vr", "auto-reply"},
		Short:   "Vacation/auto-reply management",
	}

	cmd.AddCommand(newVacationGetCmd(flags))
	cmd.AddCommand(newVacationSetCmd(flags))
	cmd.AddCommand(newVacationDisableCmd(flags))

	return cmd
}

func newVacationGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get current vacation/auto-reply settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			vr, err := client.GetVacationResponse(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get vacation response: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, vr)
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			status := "Disabled"
			if vr.IsEnabled {
				status = "Enabled"
			}
			fmt.Fprintf(tw, "Status:\t%s\n", status)

			if vr.FromDate != "" {
				fmt.Fprintf(tw, "From:\t%s\n", formatVacationDate(vr.FromDate))
			}
			if vr.ToDate != "" {
				fmt.Fprintf(tw, "To:\t%s\n", formatVacationDate(vr.ToDate))
			}
			if vr.Subject != "" {
				fmt.Fprintf(tw, "Subject:\t%s\n", vr.Subject)
			}
			tw.Flush()

			if vr.TextBody != "" {
				fmt.Println("\nMessage:")
				fmt.Println(vr.TextBody)
			}

			return nil
		},
	}

	return cmd
}

func newVacationSetCmd(flags *rootFlags) *cobra.Command {
	var subject, body, htmlBody string
	var fromDate, untilDate string
	var enable bool

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set vacation/auto-reply settings",
		Long: `Configure vacation auto-reply settings.

Dates should be in RFC3339 format (e.g., 2025-12-25T00:00:00Z) or
simple date format (YYYY-MM-DD) which will be converted to midnight UTC.

Examples:
  fastmail vacation set --enable --subject "Away" --body "I'm on vacation"
  fastmail vacation set --enable --from 2025-12-20 --until 2025-12-27 --body "Away for holidays"
  fastmail vacation set --enable --subject "Out of office" --from 2025-12-20 --body "I'll respond after the holidays"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			// Parse simple date formats to RFC3339
			if fromDate != "" {
				fromDate, err = parseVacationDate(fromDate)
				if err != nil {
					return fmt.Errorf("invalid --from date: %w", err)
				}
			}
			if untilDate != "" {
				untilDate, err = parseVacationDate(untilDate)
				if err != nil {
					return fmt.Errorf("invalid --until date: %w", err)
				}
			}

			// Warn about unsanitized HTML
			if htmlBody != "" && !isJSON(cmd.Context()) {
				fmt.Fprintln(os.Stderr, "Warning: HTML body is not sanitized. Ensure content is safe before enabling.")
			}

			opts := jmap.SetVacationResponseOpts{
				IsEnabled: enable,
				FromDate:  fromDate,
				ToDate:    untilDate,
				Subject:   subject,
				TextBody:  body,
				HTMLBody:  htmlBody,
			}

			err = client.SetVacationResponse(cmd.Context(), opts)
			if err != nil {
				return fmt.Errorf("failed to set vacation response: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"status":  "updated",
					"enabled": enable,
				})
			}

			if enable {
				fmt.Println("Vacation auto-reply enabled")
			} else {
				fmt.Println("Vacation auto-reply configured (use --enable to activate)")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&enable, "enable", false, "Enable the vacation responder")
	cmd.Flags().StringVar(&subject, "subject", "", "Auto-reply subject line")
	cmd.Flags().StringVar(&body, "body", "", "Auto-reply message body")
	cmd.Flags().StringVar(&htmlBody, "html", "", "Auto-reply HTML body (not sanitized, use with caution)")
	cmd.Flags().StringVar(&fromDate, "from", "", "Start date (YYYY-MM-DD or RFC3339)")
	cmd.Flags().StringVar(&untilDate, "until", "", "End date (YYYY-MM-DD or RFC3339)")

	return cmd
}

func newVacationDisableCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable vacation/auto-reply",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			err = client.DisableVacationResponse(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to disable vacation response: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"status":  "disabled",
					"enabled": false,
				})
			}

			fmt.Println("Vacation auto-reply disabled")
			return nil
		},
	}

	return cmd
}

// parseVacationDate parses a date string and returns RFC3339 format.
// Accepts YYYY-MM-DD (converts to midnight UTC) or full RFC3339.
func parseVacationDate(s string) (string, error) {
	// Try RFC3339 first
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return s, nil
	}

	// Try simple date format
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", fmt.Errorf("use YYYY-MM-DD or RFC3339 format")
	}

	return t.UTC().Format(time.RFC3339), nil
}

// formatVacationDate formats an RFC3339 date for display.
func formatVacationDate(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02 15:04 MST")
}
