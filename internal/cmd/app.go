package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/salmonumbrella/fastmail-cli/internal/config"
	cerrors "github.com/salmonumbrella/fastmail-cli/internal/errors"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/salmonumbrella/fastmail-cli/internal/ui"
	"github.com/salmonumbrella/fastmail-cli/internal/webdav"
	"github.com/spf13/cobra"
)

type appKey struct{}

type App struct {
	Flags  *rootFlags
	UI     *ui.UI
	Logger Logger
}

// Logger is the minimal interface we need from slog.Logger.
type Logger interface {
	Debug(msg string, args ...any)
}

func NewApp() *App {
	flags := rootFlags{
		Color:  envOr("FASTMAIL_COLOR", "auto"),
		Output: envOr("FASTMAIL_OUTPUT", "text"),
		Yes:    envBool("FASTMAIL_YES", false) || envBool("FASTMAIL_NO_INPUT", false) || envBool("FASTMAIL_NON_INTERACTIVE", false),
	}
	return &App{Flags: &flags}
}

func WithApp(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, appKey{}, app)
}

func AppFromContext(ctx context.Context) *App {
	if app, ok := ctx.Value(appKey{}).(*App); ok {
		return app
	}
	return nil
}

// runE wraps a cobra RunE to inject the App and normalize errors.
func runE(app *App, fn func(cmd *cobra.Command, args []string, app *App) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if app == nil {
			app = AppFromContext(cmd.Context())
		}
		if app == nil {
			app = &App{Flags: &rootFlags{}}
		}
		return mapCommandError(fn(cmd, args, app))
	}
}

func (a *App) IsJSON(ctx context.Context) bool {
	mode, ok := ctx.Value(outputModeKey).(outfmt.Mode)
	return ok && mode == outfmt.JSON
}

func (a *App) Query(ctx context.Context) string {
	query, _ := ctx.Value(queryKey).(string)
	return query
}

func (a *App) PrintJSON(cmd *cobra.Command, v any) error {
	return outfmt.PrintJSONFiltered(v, a.Query(cmd.Context()))
}

func (a *App) Confirm(cmd *cobra.Command, skip bool, prompt string, accepted ...string) (bool, error) {
	if skip || a.IsJSON(cmd.Context()) || (a.Flags != nil && a.Flags.Yes) {
		return true, nil
	}
	return confirmPrompt(os.Stderr, prompt, accepted...)
}

func (a *App) RequireAccount() (string, error) {
	if a.Flags != nil && a.Flags.Account != "" {
		return a.Flags.Account, nil
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

// JMAPClient creates a JMAP client for the configured account.
func (a *App) JMAPClient() (*jmap.Client, error) {
	account, err := a.RequireAccount()
	if err != nil {
		return nil, err
	}

	token, err := config.GetToken(account)
	if err != nil {
		return nil, fmt.Errorf("failed to get token for %s: %w", account, err)
	}

	return jmap.NewClient(token), nil
}

// WebDAVClient creates a WebDAV client for the configured account.
func (a *App) WebDAVClient() (*webdav.Client, error) {
	account, err := a.RequireAccount()
	if err != nil {
		return nil, err
	}

	token, err := config.GetToken(account)
	if err != nil {
		return nil, fmt.Errorf("failed to get token for %s: %w", account, err)
	}

	return webdav.NewClient(token), nil
}

// Suggest wraps an error with a user-facing suggestion.
func Suggest(err error, suggestion string) error {
	return cerrors.WithSuggestion(err, suggestion)
}
