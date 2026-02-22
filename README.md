# 📧 Fastmail CLI — Email in your terminal.

Fastmail in your terminal. Manage email, masked addresses, calendars, contacts, files, and vacation auto-replies.

## Features

- **Authentication** - credentials stored securely in keychain
- **Calendar** - manage calendars and events, send invitations
- **Contacts** - create, update, and search contacts
- **Email** - send, receive, search, and organize emails
- **Drafts** - create, list, and send draft emails
- **Files** - upload, download, and manage files via WebDAV
- **Masked email** - create disposable addresses to protect your inbox
- **Multiple accounts** - manage multiple Fastmail accounts
- **Sieve** - manage custom Sieve filters (requires browser session credentials)
- **Vacation** - set out-of-office auto-reply messages

## API Availability

Fastmail's public API provides limited access. Not all CLI commands work with standard Fastmail accounts.

### Available to All Accounts

These commands use Fastmail's standard JMAP API scopes:

| Command | Description | API Scope |
|---------|-------------|-----------|
| `email` | Send, receive, search, organize emails | `urn:ietf:params:jmap:mail` |
| `masked` | Create and manage masked email addresses | `https://www.fastmail.com/dev/maskedemail` |

### Limited Availability

These commands require additional API access that is **not available** to standard Fastmail accounts:

| Command | Description | Status |
|---------|-------------|--------|
| `calendar` | Calendars, events, invitations | Requires CalDAV/JMAP Calendar scope |
| `contacts` | Contact management | Requires CardDAV/JMAP Contacts scope |
| `vacation` | Auto-reply settings | Requires VacationResponse scope |
| `files` | File storage via WebDAV | Requires Files scope |
| `quota` | Storage quota information | Requires Quota scope |
| `sieve` | Sieve filter management | Requires browser session credentials (internal API) |

These features may become available through:
- Fastmail business/enterprise accounts
- Future API expansions
- Direct CalDAV/CardDAV access (not yet implemented in this CLI)

For the latest on API availability, see [Fastmail's developer documentation](https://www.fastmail.com/developer/).

## Installation

### Homebrew

```bash
brew install salmonumbrella/tap/fastmail-cli
```

## Quick Start

### 1. Authenticate

Choose one of two methods:

**Browser:**
```bash
fastmail auth
# (or: fastmail auth login)
```

**Terminal:**
```bash
fastmail auth add you@fastmail.com
# You'll be prompted securely for your API token
```

### 2. Test Authentication

```bash
fastmail auth status
```

## Configuration

### Account Selection

Specify the account using either a flag or environment variable:

```bash
# Via flag
fastmail email list --account you@fastmail.com

# Via environment
export FASTMAIL_ACCOUNT=you@fastmail.com
fastmail email list
```

### Environment Variables

- `FASTMAIL_ACCOUNT` - Default account email to use
- `FASTMAIL_TOKEN` - API token for `fastmail auth add` in non-interactive use
- `FASTMAIL_CREDENTIALS_DIR` - Credential storage root for keyring fallback files (`<dir>/fastmail-cli/keyring`)
- `OPENCLAW_CREDENTIALS_DIR` - Shared credentials root used when `FASTMAIL_CREDENTIALS_DIR` is not set
- `FASTMAIL_KEYRING_PASSWORD` - Password for encrypted keyring file backend (non-interactive)
- `FASTMAIL_KEYRING_BACKEND` - Keyring backend: `auto` (default), `default`, `file`, `keychain`, `wincred`, or `secret-service`
- `FASTMAIL_NO_BROWSER` - Disable auto-opening browser during `fastmail auth login`
- `FASTMAIL_OUTPUT` - Output format: `text` (default) or `json`
- `FASTMAIL_COLOR` - Color mode: `auto` (default), `always`, or `never`

OpenClaw compatibility: when present, `~/.openclaw/.env` is auto-loaded at startup.

### Non-Interactive Mode

Use `--yes` / `-y` to skip confirmation prompts (useful for agents and scripting).

Environment alternatives:
- `FASTMAIL_YES=1`
- `FASTMAIL_NON_INTERACTIVE=1`
- `FASTMAIL_NO_INPUT=1`

## Security

### Credential Storage

Credentials are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

## Commands

### Authentication

```bash
fastmail auth login                # Authenticate via browser (recommended)
fastmail auth login --no-browser   # Headless-friendly: print setup URL, don't auto-open browser
fastmail auth add <email>          # Add account manually (prompts securely)
fastmail auth list                 # List configured accounts
fastmail auth status               # Show active account
fastmail auth remove <email>       # Remove account
```

### Email

```bash
fastmail email list [--limit <n>] [--mailbox <name>]
fastmail email search <query> [--limit <n>]
fastmail email get <emailId>
fastmail email send --to <email> --subject <text> --body <text> [--cc <email>]
fastmail email move <emailId> --to <mailbox>
fastmail email mark-read <emailId> [--unread]
fastmail email delete <emailId>
fastmail email thread <threadId>
fastmail email attachments <emailId>
fastmail email download <emailId> <blobId> [output-file]
fastmail email import <file.eml>
fastmail email mailboxes
fastmail email mailbox-create <name>
fastmail email mailbox-rename <oldName> <newName>
fastmail email mailbox-delete <name>

# Bulk operations
fastmail email bulk-delete <emailId>...
fastmail email bulk-move <emailId>... --to <mailbox>
fastmail email bulk-mark-read <emailId>... [--unread]
```

### Drafts

```bash
fastmail draft list [--limit <n>]
fastmail draft get <draftId>
fastmail draft new --to <email> --subject <text> --body <text>
fastmail draft new --reply-to <emailId> --body <text>
fastmail draft send <draftId> [--yes]
fastmail draft delete <draftId> [--yes]
```

Aliases: `drafts`

### Sieve

Sieve management uses Fastmail's internal API and requires browser session credentials.

```bash
fastmail sieve auth --token <fma1-...> --cookie "__Host-s_...=..."
fastmail sieve get [--block start|middle|end|require|blocked|rules]
fastmail sieve set --start-file <file> [--middle-file <file>] [--end-file <file>]
fastmail sieve edit --block start|middle|end
```

### Masked Email

```bash
fastmail masked create <domain> [description]
fastmail masked list [domain]
fastmail masked get <email>
fastmail masked enable <email>
fastmail masked disable <email>
fastmail masked enable --domain <domain>       # Bulk enable all aliases for domain
fastmail masked disable --domain <domain>      # Bulk disable all aliases for domain
fastmail masked disable --domain <domain> --dry-run
fastmail masked description <email> <text>
fastmail masked delete <email>
```

Aliases: `mask`, `alias`

### Calendar

```bash
fastmail calendar list
fastmail calendar events [--calendar-id <id>] [--from <date>] [--to <date>]
fastmail calendar event-get <eventId>
fastmail calendar event-create --title <text> --start <datetime> --end <datetime> ...
fastmail calendar event-update <eventId> [--title <text>] [--start <datetime>] ...
fastmail calendar event-delete <eventId>
fastmail calendar invite --title <text> --start <datetime> --end <datetime> --attendees <email>...
```

### Contacts

```bash
fastmail contacts list
fastmail contacts search <query>
fastmail contacts get <contactId>
fastmail contacts create --first-name <name> --last-name <name> --email <email> ...
fastmail contacts update <contactId> [--first-name <name>] [--email <email>] ...
fastmail contacts delete <contactId>
fastmail contacts addressbooks
```

### Files

```bash
fastmail files list [path]
fastmail files upload <local-file> <remote-path>
fastmail files download <remote-path> [local-file]
fastmail files delete <remote-path>
fastmail files mkdir <remote-path>
fastmail files move <source> <destination>
```

### Vacation

```bash
fastmail vacation get
fastmail vacation set --subject <text> --body <text> [--from <date>] [--to <date>]
fastmail vacation disable
```

Aliases: `vr`, `auto-reply`

### Storage Quota

```bash
fastmail quota                     # Show quotas with human-readable sizes
fastmail quota --format bytes      # Show raw byte values
```

Aliases: `storage`, `usage`

## Output Formats

### Text

Human-readable tables with formatting:

```bash
$ fastmail email list --limit 3
ID                   FROM                    SUBJECT                   DATE
Mf123abc...          alice@example.com       Meeting tomorrow          2024-01-15 14:30
Mf456def...          bob@example.com         Invoice #2024-001         2024-01-15 12:15
Mf789ghi...          team@company.com        Weekly update             2024-01-15 10:00

$ fastmail masked list example.com
EMAIL                              STATE      DESCRIPTION
user.abc123@fastmail.com           enabled    Shopping account
user.def456@fastmail.com           disabled   Newsletter signup
```

### JSON

Machine-readable output:

```bash
$ fastmail --output json email list --limit 1
[
  {
    "id": "Mf123abc...",
    "from": {"email": "alice@example.com", "name": "Alice"},
    "subject": "Meeting tomorrow",
    "receivedAt": "2024-01-15T14:30:00Z"
  }
]
```

Data goes to stdout, errors and progress to stderr for clean piping.

## Examples

### Send an email

```bash
# Send simple email
fastmail email send \
  --to colleague@example.com \
  --subject "Project update" \
  --body "Here's the latest status..."

# Send with CC
fastmail email send \
  --to alice@example.com \
  --cc bob@example.com \
  --subject "Team sync" \
  --body "Let's discuss the roadmap"
```

### Create masked email for a service

```bash
# Create alias for shopping site
fastmail masked create shop.example.com "Amazon account"

# Later, disable all aliases for that domain
fastmail masked disable --domain shop.example.com
```

### Search and download attachments

```bash
# Search for emails with "invoice"
fastmail email search "invoice" --limit 10

# List attachments for an email
fastmail email attachments <emailId>

# Download specific attachment
fastmail email download <emailId> <blobId> invoice.pdf
```

### Organize inbox

```bash
# List mailboxes
fastmail email mailboxes

# Move email to Archive
fastmail email move <emailId> --to Archive

# Mark as read
fastmail email mark-read <emailId>
```

### Bulk email operations

```bash
# Delete multiple emails
fastmail email bulk-delete <emailId1> <emailId2> <emailId3>

# Move multiple emails to a folder
fastmail email bulk-move <emailId1> <emailId2> --to Archive

# Mark multiple emails as read
fastmail email bulk-mark-read <emailId1> <emailId2> <emailId3>
```

### Set vacation auto-reply

```bash
# Set out-of-office message
fastmail vacation set \
  --subject "Out of office" \
  --body "I'm away until Jan 20. For urgent matters, contact team@company.com" \
  --from 2024-01-15 \
  --to 2024-01-20

# Check current settings
fastmail vacation get

# Disable when back
fastmail vacation disable
```

### Create calendar invitations

```bash
# Create meeting with attendees
fastmail calendar invite \
  --title "Team standup" \
  --start "2024-01-20T10:00:00" \
  --end "2024-01-20T10:30:00" \
  --attendees alice@example.com bob@example.com
```

### Switch between accounts

```bash
# Check personal account
fastmail email list --account personal@fastmail.com

# Check work account
fastmail email list --account work@fastmail.com

# Or set default
export FASTMAIL_ACCOUNT=work@fastmail.com
fastmail email list
```

### Debug Mode

Enable verbose output for troubleshooting:

```bash
fastmail --debug email list
# Shows: API requests, responses, and internal operations
```

### Dry-Run Mode

Preview bulk operations before executing:

```bash
fastmail masked disable --domain example.com --dry-run
# Output:
# [DRY-RUN] Would disable 3 masked emails:
#   user.abc123@fastmail.com
#   user.def456@fastmail.com
#   user.ghi789@fastmail.com
# No changes made (dry-run mode)
```

## Global Flags

All commands support these flags:

- `--account <email>` - Account to use (overrides FASTMAIL_ACCOUNT)
- `--output <format>` - Output format: `text` or `json` (default: text)
- `--color <mode>` - Color mode: `auto`, `always`, or `never` (default: auto)
- `--debug` - Enable debug output (shows API operations)
- `--help` - Show help for any command
- `--version` - Show version information

## Shell Completions

Generate shell completions for your preferred shell:

### Bash

```bash
# macOS (Homebrew):
fastmail completion bash > $(brew --prefix)/etc/bash_completion.d/fastmail

# Linux:
fastmail completion bash > /etc/bash_completion.d/fastmail

# Or source directly in current session:
source <(fastmail completion bash)
```

### Zsh

```zsh
# Save to fpath:
fastmail completion zsh > "${fpath[1]}/_fastmail"

# Or add to .zshrc for auto-loading:
echo 'autoload -U compinit; compinit' >> ~/.zshrc
echo 'source <(fastmail completion zsh)' >> ~/.zshrc
```

### Fish

```fish
fastmail completion fish > ~/.config/fish/completions/fastmail.fish
```

### PowerShell

```powershell
# Load for current session:
fastmail completion powershell | Out-String | Invoke-Expression

# Or add to profile for persistence:
fastmail completion powershell >> $PROFILE
```

**Note**: Shell completions are currently disabled. To enable, set `DisableDefaultCmd: false` in `internal/cmd/root.go`.

## Development

After cloning, install git hooks:

```bash
make setup
```

This installs [lefthook](https://github.com/evilmartians/lefthook) pre-commit and pre-push hooks for linting and testing.

## License

MIT

## Links

- [Fastmail API Documentation](https://www.fastmail.com/developer/)
- [JMAP Specification](https://jmap.io/)
