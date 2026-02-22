package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// Shortcut commands are top-level desire paths for common email operations.
// They intentionally duplicate the underlying email subcommands so agents can
// succeed with a “verb-first” mental model (e.g. `fastmail search ...`).

func newSearchShortcutCmd(app *App) *cobra.Command {
	cmd := newEmailSearchCmd(app)
	cmd.Use = "search <query>"
	cmd.Aliases = []string{"find"}
	cmd.Short = "Search emails (shortcut for 'fastmail email search')"
	cmd.Long = strings.ReplaceAll(cmd.Long, "fastmail email search", "fastmail search")
	return cmd
}

func newListShortcutCmd(app *App) *cobra.Command {
	cmd := newEmailListCmd(app)
	cmd.Use = "list"
	cmd.Aliases = []string{"ls"}
	cmd.Short = "List emails (shortcut for 'fastmail email list')"
	return cmd
}

func newGetShortcutCmd(app *App) *cobra.Command {
	cmd := newEmailGetCmd(app)
	cmd.Use = "get <emailId>"
	cmd.Aliases = []string{"show", "cat"}
	cmd.Short = "Get email by ID (shortcut for 'fastmail email get')"
	return cmd
}

func newSendShortcutCmd(app *App) *cobra.Command {
	cmd := newEmailSendCmd(app)
	cmd.Use = "send"
	cmd.Aliases = []string{"compose", "new"}
	cmd.Short = "Send an email (shortcut for 'fastmail email send')"
	return cmd
}

func newThreadShortcutCmd(app *App) *cobra.Command {
	cmd := newEmailThreadCmd(app)
	cmd.Use = "thread <threadId>"
	cmd.Aliases = []string{"t"}
	cmd.Short = "Get all emails in a thread (shortcut for 'fastmail email thread')"
	return cmd
}

func newMailboxesShortcutCmd(app *App) *cobra.Command {
	cmd := newEmailMailboxesCmd(app)
	cmd.Use = "mailboxes"
	cmd.Aliases = []string{"folders"}
	cmd.Short = "List mailboxes (shortcut for 'fastmail email mailboxes')"
	return cmd
}
