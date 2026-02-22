# Email Tracking Worker

Cloudflare Worker that receives tracking pixels and records email opens.

## Deploy

```bash
cd tools/email-tracking-worker

# Install dependencies
pnpm install

# Create D1 database
wrangler d1 create email-tracker
# Copy the database_id from output

# Update wrangler.toml with the database_id
# database_id = "your-actual-id-here"

# Create KV namespace for rate limiting
wrangler kv namespace create RATE_KV
# Copy id and preview_id into wrangler.toml

# Initialize database schema
wrangler d1 execute email-tracker --file=schema.sql

# Deploy worker
wrangler deploy

# Set secrets (use the same keys in fastmail-cli setup!)
wrangler secret put TRACKING_KEY
wrangler secret put TRACKING_KEY_V1
wrangler secret put TRACKING_CURRENT_KEY_VERSION
wrangler secret put ADMIN_KEY
```

## Configure fastmail-cli

After deploying, configure fastmail-cli to use your worker:

```bash
fastmail email track setup --worker-url https://email-tracker.<your-subdomain>.workers.dev
```

Enter the same tracking key secrets and ADMIN_KEY you set as wrangler secrets.

When rotating keys, run:

```
fastmail email track rotate
```

## Rate limiting behavior

- 100 tracking-pixel hits per hour per source IP (backed by `RATE_KV`).
- Duplicate opens from the same tracking ID and IP within one hour are silently ignored.
- Pixel responses are always returned even when requests are rate-limited or deduplicated.

## Endpoints

- `GET /p/{blob}.gif` - Tracking pixel (records open, returns 1x1 GIF)
- `GET /q/{blob}` - Query opens for a specific tracking ID
- `GET /opens` - Admin endpoint to list all opens (requires Bearer token)
- `GET /health` - Health check

## Sharing with gogcli

This worker is designed to be shared between fastmail-cli and gogcli. Both CLIs store config at `~/.config/email-tracking/` so they use the same keys.
