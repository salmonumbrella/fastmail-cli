# fastmail-cli

A command-line interface for Fastmail, focused on Email and Masked Email management.

## Install

### Homebrew (macOS/Linux)

```bash
brew install salmonumbrella/tap/fastmail
```

### Go Install

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

## Setup

Run the setup wizard (opens in browser):

```bash
fastmail auth
```

This will guide you through:
1. Getting an API token from Fastmail Settings
2. Testing the connection
3. Saving credentials to your system keychain

### Manual Setup

Alternatively, add an account directly:

```bash
fastmail auth add you@fastmail.com
# Paste your API token when prompted
```

Get an API token from: Fastmail Settings > Privacy & Security > API tokens

## Usage

If you have only one account configured, commands will use it automatically. For multiple accounts, set a default:

```bash
export FASTMAIL_ACCOUNT=you@fastmail.com
```

Or use `--account you@fastmail.com` with each command. Check which account is active:

```bash
fastmail auth status
```

## Email

```bash
# List recent emails
fastmail email list --limit 10

# List emails in a specific mailbox
fastmail email list --mailbox Inbox
fastmail email list --mailbox Archive

# Search emails
fastmail email search "invoice" --limit 20

# Read an email
fastmail email get <emailId>

# Send email
fastmail email send --to someone@example.com --subject "Hi" --body "Hello"
fastmail email send --to a@example.com --cc b@example.com --subject "Hi" --body "Hello"

# Move email to folder
fastmail email move <emailId> --to Archive
fastmail email move <emailId> --to Trash

# Mark as read/unread
fastmail email mark-read <emailId>
fastmail email mark-read <emailId> --unread

# View conversation thread
fastmail email thread <threadId>

# List attachments
fastmail email attachments <emailId>

# Download attachment
fastmail email download <emailId> <blobId>
fastmail email download <emailId> <blobId> output.pdf

# Delete email (moves to trash)
fastmail email delete <emailId>

# List mailboxes (folders)
fastmail email mailboxes
```

## Masked Email

Masked emails are disposable aliases that forward to your inbox. Use them for signups to protect your real email address.

```bash
# Create alias for a domain (or get existing one)
fastmail masked create example.com
fastmail masked create example.com "Shopping account"

# List all aliases
fastmail masked list

# List aliases for a specific domain
fastmail masked list example.com

# Get alias details
fastmail masked get user.abc123@fastmail.com

# Enable/disable alias
fastmail masked enable user.abc123@fastmail.com
fastmail masked disable user.abc123@fastmail.com

# Bulk operations (all aliases for a domain)
fastmail masked enable --domain example.com
fastmail masked disable --domain example.com
fastmail masked disable --domain example.com --dry-run

# Update description
fastmail masked description user.abc123@fastmail.com "New description"

# Delete alias (emails will bounce)
fastmail masked delete user.abc123@fastmail.com
```

## Account Management

```bash
# Setup (browser wizard)
fastmail auth

# Add account manually
fastmail auth add you@fastmail.com

# List accounts
fastmail auth list

# Show default account
fastmail auth status

# Remove account
fastmail auth remove you@fastmail.com
```

## Output Formats

```bash
# Default: human-readable text
fastmail email list

# JSON for scripting
fastmail --output=json email list
fastmail --output=json email list | jq '.[0].subject'
fastmail --output=json masked list | jq '.[] | select(.state == "enabled")'
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `FASTMAIL_ACCOUNT` | Default account email |
| `FASTMAIL_OUTPUT` | Default output format (`text` or `json`) |
| `FASTMAIL_COLOR` | Color mode (`auto`, `always`, `never`) |

## Development

```bash
make build    # Build binary to ./bin/fastmail
make test     # Run tests
make fmt      # Format code
make lint     # Lint code (requires golangci-lint)
```

