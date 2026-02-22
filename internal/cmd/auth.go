package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/99designs/keyring"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/salmonumbrella/fastmail-cli/internal/auth"
	"github.com/salmonumbrella/fastmail-cli/internal/config"
	"github.com/salmonumbrella/fastmail-cli/internal/logging"
	"github.com/salmonumbrella/fastmail-cli/internal/ui"
)

const credentialWarningAge = 90 * 24 * time.Hour // 90 days

// checkCredentialAge returns a warning message if credentials are older than 90 days
func checkCredentialAge(created time.Time) string {
	if created.IsZero() {
		return ""
	}
	age := time.Since(created)
	if age > credentialWarningAge {
		days := int(age.Hours() / 24)
		return fmt.Sprintf("Warning: credentials are %d days old, consider rotating", days)
	}
	return ""
}

func newAuthCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication and account management",
		Long:  `Manage Fastmail accounts and API tokens.`,
		Args:  cobra.NoArgs,
		RunE: runE(app, func(cmd *cobra.Command, _ []string, _ *App) error {
			// Desire path: `fastmail auth` should perform the recommended login flow.
			return runAuthLogin(cmd)
		}),
	}

	cmd.PersistentFlags().Bool("no-browser", false, "Do not auto-open browser during login (print setup URL instead)")

	cmd.AddCommand(newAuthLoginCmd(app))
	cmd.AddCommand(newAuthAddCmd(app))
	cmd.AddCommand(newAuthListCmd(app))
	cmd.AddCommand(newAuthRemoveCmd(app))
	cmd.AddCommand(newAuthStatusCmd(app))

	return cmd
}

func runAuthLogin(cmd *cobra.Command) error {
	if app := AppFromContext(cmd.Context()); app != nil {
		if app.IsJSON(cmd.Context()) || (app.Flags != nil && app.Flags.Yes) {
			return Suggest(
				fmt.Errorf("auth login is interactive"),
				"Use 'fastmail auth add <email>' with FASTMAIL_TOKEN (or --token) for non-interactive setup",
			)
		}
	}

	noBrowser, err := cmd.Flags().GetBool("no-browser")
	if err != nil {
		return err
	}

	server := auth.NewSetupServer()
	result, err := server.Start(cmd.Context(), !noBrowser)
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	if result != nil && result.Email != "" {
		if app := AppFromContext(cmd.Context()); app != nil && app.IsJSON(cmd.Context()) {
			return app.PrintJSON(cmd, map[string]any{
				"status": "configured",
				"email":  result.Email,
			})
		}

		fmt.Fprintf(os.Stderr, "\nSetup complete! Account %s is now configured.\n", result.Email)
		fmt.Fprintf(os.Stderr, "Try: fastmail --account %s email list --limit 5\n", result.Email)
	}
	return nil
}

func newAuthLoginCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate via browser (recommended)",
		Long:  `Starts a local setup server for interactive authentication. Use --no-browser to print the URL without auto-launching a browser.`,
		Args:  cobra.NoArgs,
		RunE: runE(app, func(cmd *cobra.Command, _ []string, _ *App) error {
			return runAuthLogin(cmd)
		}),
	}
}

func newAuthAddCmd(app *App) *cobra.Command {
	var tokenFlag string

	cmd := &cobra.Command{
		Use:   "add <email>",
		Short: "Add a Fastmail account (prompts for API token)",
		Args:  cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			email := strings.TrimSpace(args[0])
			if email == "" {
				return fmt.Errorf("email cannot be empty")
			}

			var token string

			if tokenFlag != "" {
				// Security warning: --token flag exposes token in shell history and process listings
				if !app.IsJSON(cmd.Context()) {
					fmt.Fprintln(os.Stderr, "Warning: Using --token flag exposes your token in shell history and process listings.")
					fmt.Fprintln(os.Stderr, "Consider using FASTMAIL_TOKEN environment variable or interactive prompt instead.")
				}
				token = strings.TrimSpace(tokenFlag)
			} else if envToken := os.Getenv("FASTMAIL_TOKEN"); envToken != "" {
				// Use token from environment variable (secure scripting method)
				token = strings.TrimSpace(envToken)
			} else {
				if app.IsJSON(cmd.Context()) || (app.Flags != nil && app.Flags.Yes) {
					return Suggest(
						fmt.Errorf("FASTMAIL_TOKEN (or --token) is required in non-interactive mode"),
						"Set FASTMAIL_TOKEN and re-run (or pass --token, but --token is less safe)",
					)
				}

				// Prompt for API token securely
				fmt.Fprintf(os.Stderr, "Enter API token for %s: ", email)
				tokenBytes, err := term.ReadPassword(int(syscall.Stdin)) //nolint:unconvert // required for Windows where Stdin is uintptr
				fmt.Fprintln(os.Stderr)                                  // newline after password input
				if err != nil {
					return fmt.Errorf("failed to read token: %w", err)
				}
				token = strings.TrimSpace(string(tokenBytes))
			}

			if token == "" {
				return fmt.Errorf("token cannot be empty")
			}

			// Save to keychain
			if err := config.SaveToken(email, token); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"saved": true,
					"email": email,
				})
			}

			fmt.Fprintf(os.Stderr, "Saved API token for %s\n", email)
			return nil
		}),
	}

	cmd.Flags().StringVar(&tokenFlag, "token", "", "API token (deprecated: use FASTMAIL_TOKEN env var instead)")

	return cmd
}

func newAuthListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured accounts",
		Args:  cobra.NoArgs,
		RunE: runE(app, func(cmd *cobra.Command, _ []string, app *App) error {
			tokens, err := config.ListTokens()
			if err != nil {
				return fmt.Errorf("failed to list accounts: %w", err)
			}

			if len(tokens) == 0 {
				if app.IsJSON(cmd.Context()) {
					return app.PrintJSON(cmd, []string{})
				}
				printNoResults("No accounts configured")
				return nil
			}

			// Sort by email
			sort.Slice(tokens, func(i, j int) bool {
				return tokens[i].Email < tokens[j].Email
			})

			if app.IsJSON(cmd.Context()) {
				type account struct {
					Email     string `json:"email"`
					CreatedAt string `json:"created_at,omitempty"`
				}
				accounts := make([]account, len(tokens))
				for i, tok := range tokens {
					createdAt := ""
					if !tok.CreatedAt.IsZero() {
						createdAt = tok.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
					}
					accounts[i] = account{
						Email:     tok.Email,
						CreatedAt: createdAt,
					}
				}
				return app.PrintJSON(cmd, accounts)
			}

			for _, tok := range tokens {
				createdAt := ""
				if !tok.CreatedAt.IsZero() {
					createdAt = tok.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
				}
				fmt.Printf("%s\t%s\n", tok.Email, createdAt)
			}
			return nil
		}),
	}
}

func newAuthRemoveCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <email>",
		Short: "Remove a configured account",
		Args:  cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			email := strings.TrimSpace(args[0])
			if email == "" {
				return fmt.Errorf("email cannot be empty")
			}

			if err := config.DeleteToken(email); err != nil {
				if errors.Is(err, keyring.ErrKeyNotFound) {
					return fmt.Errorf("account not found: %s", email)
				}
				return fmt.Errorf("failed to remove account: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"deleted": true,
					"email":   email,
				})
			}

			fmt.Fprintf(os.Stderr, "Removed account: %s\n", email)
			return nil
		}),
	}
}

func newAuthStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current default account",
		Args:  cobra.NoArgs,
		RunE: runE(app, func(cmd *cobra.Command, _ []string, app *App) error {
			logger := logging.FromContext(cmd.Context())
			logger.Debug("auth status command started")

			u := ui.FromContext(cmd.Context())

			// Check for FASTMAIL_ACCOUNT environment variable
			envAccount := os.Getenv("FASTMAIL_ACCOUNT")
			logger.Debug("checking environment", "FASTMAIL_ACCOUNT", envAccount)

			// Get tokens with metadata (including created_at)
			tokens, err := config.ListTokens()
			if err != nil {
				return fmt.Errorf("failed to list accounts: %w", err)
			}
			logger.Debug("retrieved accounts", "count", len(tokens))

			if len(tokens) == 0 {
				if app.IsJSON(cmd.Context()) {
					return app.PrintJSON(cmd, map[string]any{
						"default": nil,
						"source":  "none",
					})
				}
				printNoResults("No accounts configured. Run: fastmail auth add <email>")
				return nil
			}

			// Extract emails and sort
			accounts := make([]string, len(tokens))
			tokenMap := make(map[string]config.Token)
			for i, tok := range tokens {
				accounts[i] = tok.Email
				tokenMap[tok.Email] = tok
			}
			sort.Strings(accounts)

			var defaultAccount string
			var source string

			if envAccount != "" {
				defaultAccount = envAccount
				source = "FASTMAIL_ACCOUNT"
			} else {
				defaultAccount = accounts[0]
				source = "first_account"
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"default":  defaultAccount,
					"source":   source,
					"accounts": accounts,
				})
			}

			fmt.Printf("Default account: %s (from %s)\n", defaultAccount, source)

			// Check credential age for default account
			if tok, ok := tokenMap[defaultAccount]; ok {
				if warning := checkCredentialAge(tok.CreatedAt); warning != "" {
					u.Warning(warning)
				}
			}

			fmt.Printf("Available accounts:\n")
			for _, acc := range accounts {
				marker := " "
				if acc == defaultAccount {
					marker = "*"
				}
				fmt.Printf("  %s %s\n", marker, acc)
			}
			return nil
		}),
	}
}
