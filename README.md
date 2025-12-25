# Fastmail CLI

CLI for managing Fastmail email, masked emails, calendars, contacts, and files.

## Features

- **Email management** - send, receive, search, and organize emails
- **Masked email (aliases)** - create disposable email addresses to protect your inbox
- **Calendar operations** - manage calendars and events
- **Contacts management** - create, update, and search contacts
- **File storage** - upload, download, and manage files via WebDAV
- **Vacation/auto-reply** - set out-of-office messages
- **Storage quota** - monitor account usage
- **Multiple account support** - manage multiple Fastmail accounts
- **Secure credential storage** using OS keyring (Keychain on macOS, Secret Service on Linux, Credential Manager on Windows)

## Installation

### Homebrew

```bash
brew install salmonumbrella/tap/fastmail
```

or

```bash
go install github.com/salmonumbrella/fastmail-cli/cmd/fastmail@latest
```

### Binary Download

Download pre-built binaries from [Releases](https://github.com/salmonumbrella/fastmail-cli/releases).

### Build from Source

```bash
git clone https://github.com/salmonumbrella/fastmail-cli.git
cd fastmail-cli
make build
./bin/fastmail --help
```

## Quick Start

### 1. Authenticate

Run the setup wizard (opens in browser):

```bash
fastmail auth
```

This will guide you through:
1. Getting an API token from Fastmail Settings
2. Testing the connection
3. Saving credentials to your system keychain

### 2. Set Default Account (Optional)

If you have multiple accounts, set a default:

```bash
export FASTMAIL_ACCOUNT=you@fastmail.com
```

### 3. Start Using Commands

```bash
# List recent emails
fastmail email list --limit 10

# Create a masked email
fastmail masked create example.com "Shopping account"

# Check storage usage
fastmail quota
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

- `FASTMAIL_ACCOUNT` - Default account email
- `FASTMAIL_OUTPUT` - Output format: `text` (default) or `json`
- `FASTMAIL_COLOR` - Color mode: `auto` (default), `always`, or `never`

## Security

### Credential Storage

Credentials are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

### Best Practices

- Use `fastmail auth` for interactive browser-based setup
- Use `fastmail auth add <email>` and enter API token when prompted
- Never commit API tokens to version control
- Get API tokens from: Fastmail Settings > Privacy & Security > API tokens
- Check active account with `fastmail auth status`

## Commands

### Authentication

```bash
fastmail auth                      # Setup wizard (opens browser)
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

### Masked Email (Aliases)

Masked emails are disposable aliases that forward to your inbox. Use them for signups to protect your real email address.

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

Note: Fastmail may use CalDAV instead of JMAP for calendars. If calendars are not available via JMAP, you'll receive an error.

```bash
fastmail calendar list
fastmail calendar events [--calendar-id <id>] [--from <date>] [--to <date>]
fastmail calendar event-get <eventId>
fastmail calendar event-create --title <text> --start <datetime> --end <datetime> ...
fastmail calendar event-update <eventId> [--title <text>] [--start <datetime>] ...
fastmail calendar event-delete <eventId>

# Calendar invitations (CalDAV)
fastmail calendar invite --title <text> --start <datetime> --end <datetime> --attendees <email>...
```

**Note**: Calendar invitations use CalDAV to create events with attendees. Fastmail automatically sends email invitations to all attendees when the event is created.

### Contacts

Note: Fastmail may use CardDAV instead of JMAP for contacts. If contacts are not available via JMAP, you'll receive an error.

```bash
fastmail contacts list
fastmail contacts search <query>
fastmail contacts get <contactId>
fastmail contacts create --first-name <name> --last-name <name> --email <email> ...
fastmail contacts update <contactId> [--first-name <name>] [--email <email>] ...
fastmail contacts delete <contactId>
fastmail contacts addressbooks
```

### Files (WebDAV)

Files are stored at https://myfiles.fastmail.com/

```bash
fastmail files list [path]
fastmail files upload <local-file> <remote-path>
fastmail files download <remote-path> [local-file]
fastmail files delete <remote-path>
fastmail files mkdir <remote-path>
fastmail files move <source> <destination>
```

### Vacation/Auto-Reply

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
fastmail quota --format human      # Explicitly use human-readable format
```

Aliases: `storage`, `usage`

## Output Formats

### Text

Human-readable output with formatting:

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

Machine-readable output for scripting and automation:

```bash
$ fastmail --output=json email list --limit 1
[
  {
    "id": "Mf123abc...",
    "from": {"email": "alice@example.com", "name": "Alice"},
    "subject": "Meeting tomorrow",
    "receivedAt": "2024-01-15T14:30:00Z"
  }
]

$ fastmail --output=json masked list | jq '.[] | select(.state == "enabled")'
```

Data goes to stdout for clean piping.

## Examples

### Send an email with attachment

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

# Preview changes first
fastmail masked disable --domain shop.example.com --dry-run
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

# Delete email (moves to Trash)
fastmail email delete <emailId>
```

### Bulk email operations

Process multiple emails at once:

```bash
# Delete multiple emails
fastmail email bulk-delete <emailId1> <emailId2> <emailId3>

# Move multiple emails to a folder
fastmail email bulk-move <emailId1> <emailId2> --to Archive

# Mark multiple emails as read
fastmail email bulk-mark-read <emailId1> <emailId2> <emailId3>

# Mark multiple as unread
fastmail email bulk-mark-read <emailId1> <emailId2> --unread
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

Create events with attendees (requires CalDAV):

```bash
# Create meeting with attendees
fastmail calendar invite \
  --title "Team standup" \
  --start "2024-01-20T10:00:00" \
  --end "2024-01-20T10:30:00" \
  --attendees alice@example.com bob@example.com

# Fastmail automatically sends email invitations to all attendees
```

### Manage files

```bash
# List files
fastmail files list /

# Upload file
fastmail files upload report.pdf /Documents/report.pdf

# Download file
fastmail files download /Documents/report.pdf local-report.pdf

# Create directory
fastmail files mkdir /Projects/2024
```

### Check storage usage

```bash
# View quota with progress bar
fastmail quota

# Get raw byte values for scripting
fastmail quota --format bytes --output json
```

### Work with multiple accounts

```bash
# Check personal account
fastmail email list --account personal@fastmail.com

# Check work account
fastmail email list --account work@fastmail.com

# Or set default
export FASTMAIL_ACCOUNT=work@fastmail.com
fastmail email list
```

## Advanced Features

### Debug Mode

Enable verbose output for troubleshooting:

```bash
fastmail --debug email list
# Shows: API requests, responses, and internal operations
```

### Improved Error Messages

Error messages now include context and helpful suggestions to guide you toward resolution:

```bash
# Example: Authentication error
Error: failed to authenticate with Fastmail
Context: Check your API token is valid and has not been revoked
Suggestion: Run 'fastmail auth' to re-authenticate

# Example: Resource not found
Error: email not found
Context: Email ID 'Mf123' does not exist or has been deleted
Suggestion: Verify the email ID with 'fastmail email list'
```

### Dry-Run Mode

Preview bulk operations before executing (masked email only):

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

Shell completions are currently disabled in this CLI. To enable them, the completion command needs to be re-enabled in the code.

Once enabled, generate shell completions for your preferred shell:

### Bash

```bash
fastmail completion bash > /usr/local/etc/bash_completion.d/fastmail
# Or for Linux:
fastmail completion bash > /etc/bash_completion.d/fastmail
```

### Zsh

```zsh
fastmail completion zsh > "${fpath[1]}/_fastmail"
# Or add to .zshrc:
echo 'eval "$(fastmail completion zsh)"' >> ~/.zshrc
```

### Fish

```fish
fastmail completion fish > ~/.config/fish/completions/fastmail.fish
```

### PowerShell

```powershell
fastmail completion powershell | Out-String | Invoke-Expression
# Or add to profile:
fastmail completion powershell >> $PROFILE
```

**Implementation Note**: To enable completions, remove the `DisableDefaultCmd: true` setting in `internal/cmd/root.go`:

```diff
 CompletionOptions: cobra.CompletionOptions{
-	DisableDefaultCmd: true,
+	DisableDefaultCmd: false,
 },
```

## Development

After cloning, install git hooks:

```bash
make setup
```

This installs [lefthook](https://github.com/evilmartians/lefthook) pre-commit and pre-push hooks for linting and testing.

Build and test:

```bash
make build    # Build binary to ./bin/fastmail
make test     # Run tests
make fmt      # Format code
make lint     # Lint code (requires golangci-lint)
make install  # Install to /usr/local/bin
```

## License

MIT

## Links

- [Fastmail API Documentation](https://www.fastmail.com/developer/)
- [JMAP Specification](https://jmap.io/)
- [GitHub Repository](https://github.com/salmonumbrella/fastmail-cli)
