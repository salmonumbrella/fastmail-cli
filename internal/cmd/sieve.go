package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/salmonumbrella/fastmail-cli/internal/config"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newSieveCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sieve",
		Short: "Sieve filter management (requires browser credentials)",
		Long: `Manage Sieve email filters using Fastmail's internal API.

This feature requires browser session credentials because Fastmail does not
expose Sieve management through the standard API. To set up:

1. Open Fastmail in your browser and go to Settings → Filters & Rules
2. Click "Edit custom Sieve code"
3. Open browser Developer Tools (F12) → Network tab
4. Make any edit and click Save
5. Find the request to /jmap/api and copy:
   - Authorization header value (starts with "fma1-")
   - Cookie header (the __Host-s_* cookie)
6. Run: fastmail sieve auth --token <token> --cookie <cookie>`,
	}

	cmd.AddCommand(newSieveAuthCmd(app))
	cmd.AddCommand(newSieveGetCmd(app))
	cmd.AddCommand(newSieveSetCmd(app))
	cmd.AddCommand(newSieveEditCmd(app))

	return cmd
}

func newSieveAuthCmd(app *App) *cobra.Command {
	var token, cookie string
	var remove bool

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Configure browser credentials for Sieve API",
		Long: `Store browser session credentials required for Sieve API access.

To get these credentials:
1. Open Fastmail web interface in your browser
2. Go to Settings → Filters & Rules → Edit custom Sieve code
3. Open Developer Tools (F12) → Network tab
4. Make any change and click Save
5. Find the POST request to /jmap/api
6. From Request Headers, copy:
   - Authorization: Bearer fma1-... (just the fma1-... part)
   - Cookie: __Host-s_xxxxx=yyyyy (just this cookie, not all cookies)`,
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			accountEmail, err := app.RequireAccount()
			if err != nil {
				return err
			}

			if remove {
				if err := config.DeleteSieveCredentials(accountEmail); err != nil {
					return fmt.Errorf("failed to remove sieve credentials: %w", err)
				}
				if app.IsJSON(cmd.Context()) {
					return app.PrintJSON(cmd, map[string]any{
						"status": "removed",
						"email":  accountEmail,
					})
				}
				fmt.Println("Sieve credentials removed")
				return nil
			}

			if token == "" || cookie == "" {
				return fmt.Errorf("--token and --cookie are required")
			}

			// Validate token format
			if !strings.HasPrefix(token, "fma1-") {
				return fmt.Errorf("token should start with 'fma1-' (browser session token)")
			}

			// Validate cookie format
			if !strings.Contains(cookie, "__Host-s_") {
				return fmt.Errorf("cookie should contain '__Host-s_' (session cookie)")
			}

			if err := config.SaveSieveCredentials(accountEmail, token, cookie); err != nil {
				return fmt.Errorf("failed to save sieve credentials: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"status": "saved",
					"email":  accountEmail,
				})
			}
			fmt.Println("Sieve credentials saved successfully")
			return nil
		}),
	}

	cmd.Flags().StringVar(&token, "token", "", "Browser session token (fma1-...)")
	cmd.Flags().StringVar(&cookie, "cookie", "", "Session cookie (__Host-s_...)")
	cmd.Flags().BoolVar(&remove, "remove", false, "Remove stored credentials")

	return cmd
}

func (app *App) SieveClient() (*jmap.SieveClient, error) {
	accountEmail, err := app.RequireAccount()
	if err != nil {
		return nil, err
	}

	token, cookie, err := config.GetSieveCredentials(accountEmail)
	if err != nil {
		return nil, fmt.Errorf("sieve credentials not configured; run 'fastmail sieve auth' first")
	}

	return jmap.NewSieveClientFromCredentials(token, cookie), nil
}

func newSieveGetCmd(app *App) *cobra.Command {
	var block string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get current Sieve scripts",
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.SieveClient()
			if err != nil {
				return err
			}

			blocks, err := client.GetSieveBlocks(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get sieve blocks: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, blocks)
			}

			// If specific block requested, show just that
			if block != "" {
				var content string
				switch block {
				case "start":
					content = blocks.SieveAtStart
				case "middle":
					content = blocks.SieveAtMiddle
				case "end":
					content = blocks.SieveAtEnd
				case "require":
					content = blocks.SieveRequire
				case "blocked":
					content = blocks.SieveForBlocked
				case "rules":
					content = blocks.SieveForRules
				default:
					return fmt.Errorf("unknown block: %s (use: start, middle, end, require, blocked, rules)", block)
				}
				fmt.Println(content)
				return nil
			}

			// Show all blocks with headers
			tw := outfmt.NewTabWriter()
			fmt.Fprintf(tw, "=== Require (read-only) ===\n")
			fmt.Fprintf(tw, "%s\n\n", blocks.SieveRequire)
			fmt.Fprintf(tw, "=== Start (writable) ===\n")
			fmt.Fprintf(tw, "%s\n\n", blocks.SieveAtStart)
			fmt.Fprintf(tw, "=== Blocked Senders (read-only) ===\n")
			fmt.Fprintf(tw, "%s\n\n", blocks.SieveForBlocked)
			fmt.Fprintf(tw, "=== Middle (writable) ===\n")
			fmt.Fprintf(tw, "%s\n\n", blocks.SieveAtMiddle)
			fmt.Fprintf(tw, "=== Rules (read-only) ===\n")
			fmt.Fprintf(tw, "%s\n\n", blocks.SieveForRules)
			fmt.Fprintf(tw, "=== End (writable) ===\n")
			fmt.Fprintf(tw, "%s\n", blocks.SieveAtEnd)
			tw.Flush()

			return nil
		}),
	}

	cmd.Flags().StringVar(&block, "block", "", "Show specific block: start, middle, end, require, blocked, rules")

	return cmd
}

func newSieveSetCmd(app *App) *cobra.Command {
	var startFile, middleFile, endFile string
	var start, middle, end string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update Sieve scripts",
		Long: `Update writable Sieve script blocks.

You can provide content directly via flags or from files:
  fastmail sieve set --start "# my rules"
  fastmail sieve set --start-file rules.sieve
  fastmail sieve set --middle-file middle.sieve --end-file end.sieve`,
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.SieveClient()
			if err != nil {
				return err
			}

			opts := jmap.SetSieveBlocksOpts{}

			// Handle start block
			if startFile != "" {
				content, err := os.ReadFile(startFile)
				if err != nil {
					return fmt.Errorf("failed to read start file: %w", err)
				}
				s := string(content)
				opts.SieveAtStart = &s
			} else if start != "" {
				opts.SieveAtStart = &start
			}

			// Handle middle block
			if middleFile != "" {
				content, err := os.ReadFile(middleFile)
				if err != nil {
					return fmt.Errorf("failed to read middle file: %w", err)
				}
				s := string(content)
				opts.SieveAtMiddle = &s
			} else if middle != "" {
				opts.SieveAtMiddle = &middle
			}

			// Handle end block
			if endFile != "" {
				content, err := os.ReadFile(endFile)
				if err != nil {
					return fmt.Errorf("failed to read end file: %w", err)
				}
				s := string(content)
				opts.SieveAtEnd = &s
			} else if end != "" {
				opts.SieveAtEnd = &end
			}

			if opts.SieveAtStart == nil && opts.SieveAtMiddle == nil && opts.SieveAtEnd == nil {
				return fmt.Errorf("at least one of --start, --middle, --end (or their -file variants) is required")
			}

			if err := client.SetSieveBlocks(cmd.Context(), opts); err != nil {
				return fmt.Errorf("failed to update sieve: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{"status": "updated"})
			}

			fmt.Println("Sieve scripts updated successfully")
			return nil
		}),
	}

	cmd.Flags().StringVar(&start, "start", "", "Content for start block")
	cmd.Flags().StringVar(&startFile, "start-file", "", "File containing start block content")
	cmd.Flags().StringVar(&middle, "middle", "", "Content for middle block")
	cmd.Flags().StringVar(&middleFile, "middle-file", "", "File containing middle block content")
	cmd.Flags().StringVar(&end, "end", "", "Content for end block")
	cmd.Flags().StringVar(&endFile, "end-file", "", "File containing end block content")

	return cmd
}

func newSieveEditCmd(app *App) *cobra.Command {
	var block string

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a Sieve block in $EDITOR",
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			if block == "" {
				return fmt.Errorf("--block is required (start, middle, or end)")
			}
			if block != "start" && block != "middle" && block != "end" {
				return fmt.Errorf("--block must be: start, middle, or end")
			}

			// Editing is inherently interactive (spawns $EDITOR).
			if app.IsJSON(cmd.Context()) || (app.Flags != nil && app.Flags.Yes) {
				return Suggest(
					fmt.Errorf("sieve edit is interactive"),
					"Use 'fastmail sieve set --start-file/--middle-file/--end-file' for non-interactive updates",
				)
			}

			client, err := app.SieveClient()
			if err != nil {
				return err
			}

			blocks, err := client.GetSieveBlocks(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get sieve blocks: %w", err)
			}

			var currentContent string
			switch block {
			case "start":
				currentContent = blocks.SieveAtStart
			case "middle":
				currentContent = blocks.SieveAtMiddle
			case "end":
				currentContent = blocks.SieveAtEnd
			}

			// Create temp file with .sieve extension for syntax highlighting
			tmpFile, err := os.CreateTemp("", "sieve-*.sieve")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			tmpPath := tmpFile.Name()
			defer func() { _ = os.Remove(tmpPath) }()

			if _, writeErr := tmpFile.WriteString(currentContent); writeErr != nil {
				_ = tmpFile.Close()
				return fmt.Errorf("failed to write temp file: %w", writeErr)
			}
			if closeErr := tmpFile.Close(); closeErr != nil {
				return fmt.Errorf("failed to close temp file: %w", closeErr)
			}

			if editErr := runEditor(tmpPath); editErr != nil {
				return editErr
			}

			updatedContent, err := os.ReadFile(tmpPath)
			if err != nil {
				return fmt.Errorf("failed to read updated file: %w", err)
			}

			opts := jmap.SetSieveBlocksOpts{}
			s := string(updatedContent)
			switch block {
			case "start":
				opts.SieveAtStart = &s
			case "middle":
				opts.SieveAtMiddle = &s
			case "end":
				opts.SieveAtEnd = &s
			}

			if err := client.SetSieveBlocks(cmd.Context(), opts); err != nil {
				return fmt.Errorf("failed to update sieve: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{"status": "updated"})
			}

			fmt.Println("Sieve scripts updated successfully")
			return nil
		}),
	}

	cmd.Flags().StringVar(&block, "block", "", "Block to edit: start, middle, or end")
	_ = cmd.MarkFlagRequired("block")

	return cmd
}

func runEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	fields := strings.Fields(editor)
	if len(fields) == 0 {
		return fmt.Errorf("invalid editor command")
	}

	cmd := exec.Command(fields[0], append(fields[1:], path)...) //nolint:gosec // Intentional: $EDITOR is user-controlled
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}
	return nil
}
