import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fetchWithRetry } from "./http.js";
import { genDeviceInfo, genGps, loadRajikoAppKey } from "./rajiko-derived.js";

/**
 * Source map in this file:
 * - retrieveToken: rajiko/modules/auth.js (retrieve_token) + rajiko/background.js (auth headers behavior)
 */
interface TokenCacheItem {
  token: string;
  requestTime: number;
}

const tokenCache = new Map<string, TokenCacheItem>();
let cacheLoaded = false;
const AUTH_CACHE_PATH = path.resolve(".cache", "auth-tokens.json");

/**
 * Source: rajiko/modules/auth.js -> retrieve_token
 * Adaptation: area is provided by caller, and storage/session dependency is replaced by persistent file cache.
 */
export async function retrieveToken(areaId: string): Promise<string> {
  await ensureCacheLoaded();
  const cached = tokenCache.get(areaId);
  if (cached && Date.now() - cached.requestTime < 42e5) {
    return cached.token;
  }

  const keyMaterial = await loadRajikoAppKey();
  const info = genDeviceInfo(keyMaterial.appVersion);

  const auth1 = await fetchWithRetry(
    "https://radiko.jp/v2/api/auth1",
    {
      headers: {
        "X-Radiko-App": keyMaterial.appId,
        "X-Radiko-App-Version": info.appVersion,
        "X-Radiko-Device": info.device,
        "X-Radiko-User": info.userId,
      },
    },
    { retries: 3, delayMs: 300 },
  );
  if (!auth1.ok) {
    throw new Error(`auth1 failed: ${auth1.status}`);
  }

  const token = auth1.headers.get("x-radiko-authtoken");
  const offsetStr = auth1.headers.get("x-radiko-keyoffset");
  const lengthStr = auth1.headers.get("x-radiko-keylength");
  if (!token || !offsetStr || !lengthStr) {
    throw new Error("auth1 response is missing token or key range headers");
  }

  const offset = Number(offsetStr);
  const length = Number(lengthStr);
  const fullKey = Buffer.from(keyMaterial.appKeyBase64, "base64");
  const partial = fullKey.subarray(offset, offset + length).toString("base64");

  const auth2 = await fetchWithRetry(
    "https://radiko.jp/v2/api/auth2",
    {
      headers: {
        "X-Radiko-App": keyMaterial.appId,
        "X-Radiko-App-Version": info.appVersion,
        "X-Radiko-Device": info.device,
        "X-Radiko-User": info.userId,
        "X-Radiko-AuthToken": token,
        "X-Radiko-Partialkey": partial,
        "X-Radiko-Location": genGps(areaId),
        "X-Radiko-Connection": "wifi",
        "User-Agent": info.userAgent,
      },
    },
    { retries: 3, delayMs: 300 },
  );
  if (auth2.status !== 200) {
    throw new Error(`auth2 failed: ${auth2.status}`);
  }

  tokenCache.set(areaId, { token, requestTime: Date.now() });
  await saveCache();
  return token;
}

async function ensureCacheLoaded(): Promise<void> {
  if (cacheLoaded) {
    return;
  }
  cacheLoaded = true;
  try {
    const raw = await readFile(AUTH_CACHE_PATH, "utf-8");
    const json = JSON.parse(raw) as Record<string, TokenCacheItem>;
    for (const [areaId, item] of Object.entries(json)) {
      if (item?.token && Number.isFinite(item.requestTime)) {
        tokenCache.set(areaId, {
          token: item.token,
          requestTime: item.requestTime,
        });
      }
    }
  } catch {
    // Ignore missing/broken cache files and continue with fresh auth.
  }
}

async function saveCache(): Promise<void> {
  const dir = path.dirname(AUTH_CACHE_PATH);
  await mkdir(dir, { recursive: true });
  const obj: Record<string, TokenCacheItem> = {};
  for (const [k, v] of tokenCache.entries()) {
    obj[k] = v;
  }
  await writeFile(AUTH_CACHE_PATH, JSON.stringify(obj, null, 2), "utf-8");
}
