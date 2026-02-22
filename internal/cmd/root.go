package cmd

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strings"

	cerrors "github.com/salmonumbrella/fastmail-cli/internal/errors"
	"github.com/salmonumbrella/fastmail-cli/internal/logging"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/salmonumbrella/fastmail-cli/internal/ui"
	"github.com/spf13/cobra"
)

//go:embed help.txt
var helpText string

// Version information - set at build time via ldflags
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

type rootFlags struct {
	Color          string
	Account        string
	Output         string
	Debug          bool
	Query          string
	Yes            bool
	NoInput        bool
	NonInteractive bool
}

type contextKey string

const (
	outputModeKey contextKey = "outputMode"
	queryKey      contextKey = "query"
)

func Execute(args []string) error {
	if !isTestBinary() {
		// Best-effort OpenClaw compatibility: auto-load ~/.openclaw/.env.
		_ = loadOpenClawEnvIfPresent()
	}

	app := NewApp()
	root := NewRootCmd(app)
	root.SetArgs(args)

	err := root.Execute()
	if err != nil {
		if app.Flags.Output == "json" {
			payload := map[string]any{
				"error": map[string]any{
					"message": err.Error(),
				},
			}
			if cerrors.ContainsSuggestion(err) {
				payload["error"].(map[string]any)["suggestion"] = cerrors.GetSuggestion(err)
			}
			_ = outfmt.WriteJSON(os.Stderr, payload)
		} else {
			// Print the main error
			fmt.Fprintln(os.Stderr, "Error:", err)

			// Print suggestion if available
			if cerrors.ContainsSuggestion(err) {
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, "Suggestion:", cerrors.GetSuggestion(err))
			}
		}
	}
	return err
}

func NewRootCmd(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:           "fastmail",
		Short:         "Fastmail CLI for Email and Masked Email",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false,
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// UI (must come first)
			u := ui.New(app.Flags.Color)
			ctx := ui.WithUI(cmd.Context(), u)
			app.UI = u

			// Output format
			mode := outfmt.Text
			if app.Flags.Output == "json" {
				mode = outfmt.JSON
			}
			ctx = context.WithValue(ctx, outputModeKey, mode)

			// Query filter
			ctx = context.WithValue(ctx, queryKey, app.Flags.Query)

			// Non-interactive aliases
			if app.Flags.NoInput || app.Flags.NonInteractive {
				app.Flags.Yes = true
			}

			// Logging
			logger := logging.Setup(app.Flags.Debug)
			ctx = logging.WithLogger(ctx, logger)
			app.Logger = logger

			ctx = WithApp(ctx, app)
			cmd.SetContext(ctx)
			return nil
		},
	}
	root.PersistentFlags().StringVar(&app.Flags.Color, "color", app.Flags.Color, "Color output: auto|always|never")
	root.PersistentFlags().StringVar(&app.Flags.Account, "account", envOr("FASTMAIL_ACCOUNT", ""), "Account email for API commands")
	root.PersistentFlags().StringVar(&app.Flags.Output, "output", app.Flags.Output, "Output format: text|json")
	root.PersistentFlags().BoolVar(&app.Flags.Debug, "debug", false, "Enable debug logging")
	root.PersistentFlags().StringVar(&app.Flags.Query, "query", "", "JQ filter expression for JSON output")
	root.PersistentFlags().BoolVarP(&app.Flags.Yes, "yes", "y", false, "Skip confirmation prompts (non-interactive)")
	root.PersistentFlags().BoolVar(&app.Flags.NoInput, "no-input", false, "Alias for --yes (non-interactive)")
	root.PersistentFlags().BoolVar(&app.Flags.NonInteractive, "non-interactive", false, "Alias for --yes (non-interactive)")
	_ = root.PersistentFlags().MarkHidden("no-input")
	_ = root.PersistentFlags().MarkHidden("non-interactive")

	root.AddCommand(newAuthCmd(app))
	root.AddCommand(newEmailCmd(app))
	root.AddCommand(newMaskedCmd(app))
	root.AddCommand(newVacationCmd(app))
	root.AddCommand(newContactsCmd(app))
	root.AddCommand(newCalendarCmd(app))
	root.AddCommand(newQuotaCmd(app))
	root.AddCommand(newFilesCmd(app))
	root.AddCommand(newSieveCmd(app))
	root.AddCommand(newDraftCmd(app))

	// Desire paths: top-level shortcuts for common email workflows.
	root.AddCommand(newSearchShortcutCmd(app))
	root.AddCommand(newListShortcutCmd(app))
	root.AddCommand(newGetShortcutCmd(app))
	root.AddCommand(newSendShortcutCmd(app))
	root.AddCommand(newThreadShortcutCmd(app))
	root.AddCommand(newMailboxesShortcutCmd(app))

	// Override root help only; subcommands keep Cobra's default.
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd.Name() == root.Name() && !cmd.HasParent() {
			fmt.Print(helpText)
			return
		}
		defaultHelp(cmd, args)
	})

	return root
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func isTestBinary() bool {
	return flag.Lookup("test.v") != nil
}

// outputModeKey and queryKey are context keys for output formatting.
