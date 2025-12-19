package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/salmonumbrella/fastmail-cli/internal/config"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/logging"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/salmonumbrella/fastmail-cli/internal/ui"
)

// Version information - set at build time via ldflags
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

type rootFlags struct {
	Color   string
	Account string
	Output  string
	Debug   bool
}

type contextKey string

const (
	outputModeKey contextKey = "outputMode"
)

func Execute(args []string) error {
	flags := rootFlags{
		Color:  envOr("FASTMAIL_COLOR", "auto"),
		Output: envOr("FASTMAIL_OUTPUT", "text"),
	}

	root := &cobra.Command{
		Use:           "fastmail",
		Short:         "Fastmail CLI for Email and Masked Email",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		Example: strings.TrimSpace(`
  # One-time setup (opens browser)
  fastmail auth

  # Set default account to avoid --account flag
  export FASTMAIL_ACCOUNT=you@fastmail.com

  # Email
  fastmail email list --limit 10
  fastmail email search 'invoice' --limit 20
  fastmail email get <emailId>
  fastmail email send --to someone@example.com --subject "Hi" --body "Hello"

  # Masked Email (aliases)
  fastmail masked create example.com "Shopping account"
  fastmail masked list example.com
  fastmail masked enable user.1234@fastmail.com
  fastmail masked disable user.1234@fastmail.com

  # JSON output for scripting
  fastmail --output=json email list | jq .
`),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// UI (must come first)
			u := ui.New(flags.Color)
			ctx := ui.WithUI(cmd.Context(), u)

			// Output format
			mode := outfmt.Text
			if flags.Output == "json" {
				mode = outfmt.JSON
			}
			ctx = context.WithValue(ctx, outputModeKey, mode)

			// Logging
			logger := logging.Setup(flags.Debug)
			ctx = logging.WithLogger(ctx, logger)

			cmd.SetContext(ctx)
			return nil
		},
	}

	root.SetArgs(args)
	root.PersistentFlags().StringVar(&flags.Color, "color", flags.Color, "Color output: auto|always|never")
	root.PersistentFlags().StringVar(&flags.Account, "account", envOr("FASTMAIL_ACCOUNT", ""), "Account email for API commands")
	root.PersistentFlags().StringVar(&flags.Output, "output", flags.Output, "Output format: text|json")
	root.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "Enable debug logging")

	root.AddCommand(newAuthCmd())
	root.AddCommand(newEmailCmd(&flags))
	root.AddCommand(newMaskedCmd(&flags))
	root.AddCommand(newVacationCmd(&flags))
	root.AddCommand(newContactsCmd(&flags))
	root.AddCommand(newCalendarCmd(&flags))

	err := root.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return err
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireAccount(flags *rootFlags) (string, error) {
	if flags.Account != "" {
		return flags.Account, nil
	}

	// Auto-select primary/only account when not explicitly specified
	primary, err := config.GetPrimaryAccount()
	if err != nil {
		return "", fmt.Errorf("failed to get accounts: %w", err)
	}
	if primary != "" {
		return primary, nil
	}

	return "", fmt.Errorf("no accounts configured: run 'fastmail auth' to set up an account")
}

func isJSON(ctx context.Context) bool {
	mode, ok := ctx.Value(outputModeKey).(outfmt.Mode)
	return ok && mode == outfmt.JSON
}

// getClient creates a JMAP client for the configured account.
// It retrieves the account from flags (or FASTMAIL_ACCOUNT env var)
// and fetches the API token from the keychain.
func getClient(flags *rootFlags) (*jmap.Client, error) {
	account, err := requireAccount(flags)
	if err != nil {
		return nil, err
	}

	token, err := config.GetToken(account)
	if err != nil {
		return nil, fmt.Errorf("failed to get token for %s: %w", account, err)
	}

	return jmap.NewClient(token), nil
}
