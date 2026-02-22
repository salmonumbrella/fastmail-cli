import type { PixelPayload } from './types';

const ALGORITHM = 'AES-GCM';
const IV_LENGTH = 12;
const VERSION_BYTE_LENGTH = 1;

export async function importKey(base64Key: string): Promise<CryptoKey> {
  const keyBytes = Uint8Array.from(atob(base64Key), c => c.charCodeAt(0));
  return crypto.subtle.importKey(
    'raw',
    keyBytes,
    { name: ALGORITHM },
    false,
    ['encrypt', 'decrypt']
  );
}

export async function decrypt(blob: string, keyBase64: string): Promise<PixelPayload> {
  return decryptWithKeys(blob, { 1: keyBase64 });
}

export async function decryptWithKeys(
  blob: string,
  keysByVersion: Record<number, string>,
  currentVersion?: number
): Promise<PixelPayload> {
  const combined = decodeBlob(blob);
  let lastErr: unknown = null;

  const versionsToTry = sortedVersions(keysByVersion, combined, currentVersion);
  for (const version of versionsToTry) {
    const keyBase64 = keysByVersion[version];
    if (!keyBase64) {
      continue;
    }

    try {
      return await decryptWithVersionedPayload(combined, keyBase64, version);
    } catch (err) {
      lastErr = err;
    }
  }

  const legacyKeys = new Set<string>(Object.values(keysByVersion));
  for (const keyBase64 of legacyKeys) {
    try {
      return await decryptWithoutVersion(combined, keyBase64);
    } catch (err) {
      lastErr = err;
    }
  }

  if (lastErr instanceof Error) {
    throw lastErr;
  }
  throw new Error('unable to decrypt tracking payload');
}

export async function encrypt(payload: PixelPayload, keyBase64: string, version = 1): Promise<string> {
  const key = await importKey(keyBase64);
  const iv = crypto.getRandomValues(new Uint8Array(IV_LENGTH));
  const encoded = new TextEncoder().encode(JSON.stringify(payload));

  const ciphertext = await crypto.subtle.encrypt(
    { name: ALGORITHM, iv },
    key,
    encoded
  );

  const combined = new Uint8Array(VERSION_BYTE_LENGTH + IV_LENGTH + ciphertext.byteLength);
  combined[0] = version;
  combined.set(iv, VERSION_BYTE_LENGTH);
  combined.set(new Uint8Array(ciphertext), VERSION_BYTE_LENGTH + IV_LENGTH);

  // URL-safe base64 encode
  const base64 = btoa(String.fromCharCode(...combined));
  return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

async function decryptWithVersionedPayload(
  combined: Uint8Array,
  keyBase64: string,
  version: number
): Promise<PixelPayload> {
  if (combined.length < VERSION_BYTE_LENGTH + IV_LENGTH) {
    throw new Error('blob too short');
  }

  if (combined[0] !== version) {
    throw new Error('version mismatch');
  }

  const key = await importKey(keyBase64);
  const iv = combined.slice(1, VERSION_BYTE_LENGTH + IV_LENGTH);
  const ciphertext = combined.slice(VERSION_BYTE_LENGTH + IV_LENGTH);

  return decryptPayload( key, iv, ciphertext );
}

async function decryptWithoutVersion(
  combined: Uint8Array,
  keyBase64: string
): Promise<PixelPayload> {
  if (combined.length < IV_LENGTH) {
    throw new Error('blob too short');
  }

  const key = await importKey(keyBase64);
  const iv = combined.slice(0, IV_LENGTH);
  const ciphertext = combined.slice(IV_LENGTH);

  return decryptPayload(key, iv, ciphertext);
}

async function decryptPayload(
  key: CryptoKey,
  iv: Uint8Array,
  ciphertext: Uint8Array
): Promise<PixelPayload> {
  const decrypted = await crypto.subtle.decrypt(
    { name: ALGORITHM, iv },
    key,
    ciphertext
  );

  const text = new TextDecoder().decode(decrypted);
  return JSON.parse(text) as PixelPayload;
}

function sortedVersions(
  keysByVersion: Record<number, string>,
  combined: Uint8Array,
  currentVersion?: number
): number[] {
  const versionSet = new Map<number, boolean>();

  if (currentVersion && keysByVersion[currentVersion]) {
    versionSet.set(currentVersion, true);
  }

  if (combined.length > VERSION_BYTE_LENGTH && keysByVersion[combined[0]]) {
    versionSet.set(combined[0], true);
  }

  for (const key of Object.keys(keysByVersion)) {
    const version = Number(key);
    if (Number.isInteger(version) && version > 0) {
      versionSet.set(version, true);
    }
  }

  const versions: number[] = [];
  for (const version of versionSet.keys()) {
    versions.push(version);
  }
  versions.sort((a, b) => a - b);
  return versions;
}

function decodeBlob(blob: string): Uint8Array {
  const base64 = blob.replace(/-/g, '+').replace(/_/g, '/');
  const padded = base64 + '='.repeat((4 - base64.length % 4) % 4);
  return Uint8Array.from(atob(padded), c => c.charCodeAt(0));
}
