import type { Env, PixelPayload } from './types';
import { decryptWithKeys } from './crypto';
import { detectBot } from './bot';
import { pixelResponse } from './pixel';

const RATE_LIMIT_WINDOW_SECONDS = 60 * 60;
const RATE_LIMIT_MAX_REQUESTS = 100;
const DEDUPE_WINDOW_SECONDS = 60 * 60;

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const path = url.pathname;

    try {
      // Pixel endpoint: GET /p/:blob.gif
      if (path.startsWith('/p/') && path.endsWith('.gif')) {
        return await handlePixel(request, env, path);
      }

      // Query endpoint: GET /q/:blob
      if (path.startsWith('/q/')) {
        return await handleQuery(request, env, path);
      }

      // Admin opens endpoint: GET /opens
      if (path === '/opens') {
        return await handleAdminOpens(request, env, url);
      }

      // Health check
      if (path === '/health') {
        return new Response('ok', { status: 200 });
      }

      return new Response('Not Found', { status: 404 });
    } catch (error) {
      console.error('Handler error:', error);
      return new Response('Internal Error', { status: 500 });
    }
  },
};

async function handlePixel(request: Request, env: Env, path: string): Promise<Response> {
  // Extract blob from /p/:blob.gif
  const blob = path.slice(3, -4); // Remove '/p/' and '.gif'

  const ip = request.headers.get('CF-Connecting-IP') || 'unknown';

  if (await isRateLimited(env, ip)) {
    // Always return a valid pixel response for rate-limited traffic to avoid breaking email rendering.
    return pixelResponse();
  }

  const trackingKeys = getTrackingKeysFromEnv(env);
  const currentVersion = parseInt((env.TRACKING_CURRENT_KEY_VERSION || '').trim(), 10);

  let payload: PixelPayload;
  try {
    payload = await decryptWithKeys(
      blob,
      trackingKeys,
      Number.isNaN(currentVersion) ? undefined : currentVersion
    );
  } catch {
    // Still return pixel even if decryption fails (don't break email display)
    return pixelResponse();
  }

  // Get request metadata
  const userAgent = request.headers.get('User-Agent') || 'unknown';
  const cf = (request as any).cf || {};

  if (await isDuplicateOpen(env, blob, ip)) {
    return pixelResponse();
  }

  // Calculate time since delivery
  const now = Date.now();
  const sentAt = payload.t * 1000; // Convert to ms
  const timeSinceDelivery = now - sentAt;

  // Detect bots
  const { isBot, botType } = detectBot(userAgent, ip, timeSinceDelivery);

  const openedAt = new Date().toISOString();

  // Log to D1
  await env.DB.prepare(`
    INSERT INTO opens (
      tracking_id, recipient, subject_hash, sent_at, opened_at,
      ip, user_agent, country, region, city, timezone,
      is_bot, bot_type
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  `).bind(
    blob,
    payload.r,
    payload.s,
    new Date(sentAt).toISOString(),
    openedAt,
    ip,
    userAgent,
    cf.country || null,
    cf.region || null,
    cf.city || null,
    cf.timezone || null,
    isBot ? 1 : 0,
    botType
  ).run();

  return pixelResponse();
}

// Best-effort rate limiting using KV. Concurrent requests from the same IP may
// slightly exceed the limit due to the non-atomic read-then-write pattern.
// Uses fixed hourly windows so the TTL is not reset on each increment.
async function isRateLimited(env: Env, ip: string): Promise<boolean> {
  if (!env.RATE_KV || ip === 'unknown') {
    return false;
  }

  const hourBucket = Math.floor(Date.now() / (RATE_LIMIT_WINDOW_SECONDS * 1000));
  const rateKey = `rate:${ip}:${hourBucket}`;
  const existing = await env.RATE_KV.get(rateKey);
  const parsed = parseInt(existing || '0', 10);
  const count = Number.isNaN(parsed) ? 0 : Math.max(parsed, 0);

  if (count >= RATE_LIMIT_MAX_REQUESTS) {
    return true;
  }

  await env.RATE_KV.put(rateKey, String(count + 1), {
    expirationTtl: RATE_LIMIT_WINDOW_SECONDS,
  });

  return false;
}

async function isDuplicateOpen(env: Env, trackingId: string, ip: string): Promise<boolean> {
  if (ip === 'unknown') {
    return false;
  }

  const cutoff = new Date(Date.now() - DEDUPE_WINDOW_SECONDS * 1000).toISOString();
  const result = await env.DB.prepare(`
    SELECT 1
    FROM opens
    WHERE tracking_id = ?
      AND ip = ?
      AND opened_at > ?
    LIMIT 1
  `).bind(trackingId, ip, cutoff).first();

  return result !== null;
}

async function handleQuery(request: Request, env: Env, path: string): Promise<Response> {
  const blob = path.slice(3); // Remove '/q/'

  const trackingKeys = getTrackingKeysFromEnv(env);
  const currentVersion = parseInt((env.TRACKING_CURRENT_KEY_VERSION || '').trim(), 10);

  let payload: PixelPayload;
  try {
    payload = await decryptWithKeys(
      blob,
      trackingKeys,
      Number.isNaN(currentVersion) ? undefined : currentVersion
    );
  } catch {
    return new Response('Invalid tracking ID', { status: 400 });
  }

  const result = await env.DB.prepare(`
    SELECT
      opened_at, ip, city, region, country, timezone, is_bot, bot_type
    FROM opens
    WHERE tracking_id = ?
    ORDER BY opened_at ASC
  `).bind(
    blob
  ).all();

  const opens = result.results.map((row: any) => ({
    at: row.opened_at,
    is_bot: row.is_bot === 1,
    bot_type: row.bot_type,
    location: row.city ? {
      city: row.city,
      region: row.region,
      country: row.country,
      timezone: row.timezone,
    } : null,
  }));

  const humanOpens = opens.filter((o: any) => !o.is_bot);

  return Response.json({
    tracking_id: blob,
    recipient: payload.r,
    sent_at: new Date(payload.t * 1000).toISOString(),
    opens,
    total_opens: opens.length,
    human_opens: humanOpens.length,
    first_human_open: humanOpens[0] || null,
  });
}

async function handleAdminOpens(request: Request, env: Env, url: URL): Promise<Response> {
  // Verify admin key
  const authHeader = request.headers.get('Authorization');
  if (!authHeader || authHeader !== `Bearer ${env.ADMIN_KEY}`) {
    return new Response('Unauthorized', { status: 401 });
  }

  const recipient = url.searchParams.get('recipient');
  const since = url.searchParams.get('since');
  const limit = parseInt(url.searchParams.get('limit') || '100', 10);

  let query = 'SELECT * FROM opens WHERE 1=1';
  const params: any[] = [];

  if (recipient) {
    query += ' AND recipient = ?';
    params.push(recipient);
  }

  if (since) {
    query += ' AND opened_at >= ?';
    params.push(since);
  }

  query += ' ORDER BY opened_at DESC LIMIT ?';
  params.push(limit);

  const result = await env.DB.prepare(query).bind(...params).all();

  return Response.json({
    opens: result.results.map((row: any) => ({
      tracking_id: row.tracking_id,
      recipient: row.recipient,
      subject_hash: row.subject_hash,
      sent_at: row.sent_at,
      opened_at: row.opened_at,
      is_bot: row.is_bot === 1,
      bot_type: row.bot_type,
      location: row.city ? {
        city: row.city,
        region: row.region,
        country: row.country,
      } : null,
    })),
  });
}

function getTrackingKeysFromEnv(env: Env): Record<number, string> {
  const keys: Record<number, string> = {};

  if (env.TRACKING_KEY) {
    keys[1] = env.TRACKING_KEY;
  }

  const envRecord = env as unknown as Record<string, unknown>;
  const versionedKeyPattern = /^TRACKING_KEY_V(\d+)$/;
  for (const [key, value] of Object.entries(envRecord)) {
    const match = versionedKeyPattern.exec(key);
    if (!match) {
      continue;
    }

    const version = Number.parseInt(match[1], 10);
    if (Number.isNaN(version) || version <= 0) {
      continue;
    }

    if (typeof value === 'string') {
      keys[version] = value;
    }
  }

  return keys;
}
