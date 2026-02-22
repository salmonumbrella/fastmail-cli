package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/dateparse"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newVacationCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vacation",
		Aliases: []string{"vr", "auto-reply"},
		Short:   "Vacation/auto-reply management",
	}

	cmd.AddCommand(newVacationGetCmd(app))
	cmd.AddCommand(newVacationSetCmd(app))
	cmd.AddCommand(newVacationDisableCmd(app))

	return cmd
}

func newVacationGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get current vacation/auto-reply settings",
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			vr, err := client.GetVacationResponse(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get vacation response: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, vr)
			}

			tw := outfmt.NewTabWriter()
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
		}),
	}

	return cmd
}

func newVacationSetCmd(app *App) *cobra.Command {
	var subject, body, htmlBody string
	var fromDate, untilDate string
	var enable bool

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set vacation/auto-reply settings",
		Long: `Configure vacation auto-reply settings.

Dates should be in RFC3339 format (e.g., 2025-12-25T00:00:00Z),
simple date format (YYYY-MM-DD), or relative expressions like yesterday, 2h ago, or monday.

Examples:
  fastmail vacation set --enable --subject "Away" --body "I'm on vacation"
  fastmail vacation set --enable --from 2025-12-20 --until 2025-12-27 --body "Away for holidays"
  fastmail vacation set --enable --subject "Out of office" --from 2025-12-20 --body "I'll respond after the holidays"`,
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
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
			if htmlBody != "" && !app.IsJSON(cmd.Context()) {
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

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
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
		}),
	}

	cmd.Flags().BoolVar(&enable, "enable", false, "Enable the vacation responder")
	cmd.Flags().StringVar(&subject, "subject", "", "Auto-reply subject line")
	cmd.Flags().StringVar(&body, "body", "", "Auto-reply message body")
	cmd.Flags().StringVar(&htmlBody, "html", "", "Auto-reply HTML body (not sanitized, use with caution)")
	cmd.Flags().StringVar(&fromDate, "from", "", "Start date (RFC3339, YYYY-MM-DD, or relative like yesterday, 2h ago, monday)")
	cmd.Flags().StringVar(&untilDate, "until", "", "End date (RFC3339, YYYY-MM-DD, or relative like yesterday, 2h ago, monday)")

	return cmd
}

func newVacationDisableCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable vacation/auto-reply",
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			err = client.DisableVacationResponse(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to disable vacation response: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"status":  "disabled",
					"enabled": false,
				})
			}

			fmt.Println("Vacation auto-reply disabled")
			return nil
		}),
	}

	return cmd
}

// parseVacationDate parses a date string and returns RFC3339 format.
// Accepts RFC3339, YYYY-MM-DD (converted to midnight UTC), or relative expressions.
func parseVacationDate(s string) (string, error) {
	// Try RFC3339 first
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return s, nil
	}

	t, err := dateparse.ParseDateTimeNow(s)
	if err != nil {
		return "", fmt.Errorf("use RFC3339, YYYY-MM-DD, or relative expressions like yesterday, 2h ago, monday")
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
