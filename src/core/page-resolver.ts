import { fetchWithRetry } from "./http.js";
import { classifyRadikoLink, pickLatestDetailUrl } from "./link.js";

/**
 * Source map in this file:
 * - resolveToDetailUrl: inspired by rajiko/pages/popup.js timeshift URL detection, adapted for CLI search API
 */
/**
 * Source inspiration: rajiko/pages/popup.js (timeshift link detection from page state)
 * Adaptation: resolve latest detail URL by using Radiko search API directly.
 */
export async function resolveToDetailUrl(url: string): Promise<string> {
  const kind = classifyRadikoLink(url);
  if (kind === "detail") {
    return url;
  }
  if (kind !== "search") {
    throw new Error(`Unsupported link: ${url}`);
  }

  const apiLinks = await fetchDetailLinksFromSearchApi(url);
  if (apiLinks.length === 0) {
    throw new Error(`No detail links found in search page: ${url}`);
  }
  return pickLatestDetailUrl(apiLinks);
}

export async function fetchDetailLinksFromSearchApi(
  url: string,
): Promise<string[]> {
  const key = extractSearchKeyFromUrl(url);
  if (!key) {
    return [];
  }

  const searchUrl = new URL(
    "https://api.annex-cf.radiko.jp/v1/programs/legacy/perl/program/search",
  );
  searchUrl.searchParams.set("key", key);
  searchUrl.searchParams.set("filter", "");
  searchUrl.searchParams.set("start_day", "");
  searchUrl.searchParams.set("end_day", "");
  searchUrl.searchParams.set("area_id", "");
  searchUrl.searchParams.set("cur_area_id", "");
  searchUrl.searchParams.set("uid", randomUid());
  searchUrl.searchParams.set("row_limit", "12");
  searchUrl.searchParams.set("app_id", "pc");
  searchUrl.searchParams.set("action_id", "0");

  try {
    const response = await fetchWithRetry(searchUrl.toString(), undefined, {
      retries: 3,
      delayMs: 300,
    });
    if (!response.ok) {
      return [];
    }
    const json = (await response.json()) as unknown;
    return buildDetailUrlsFromSearchApiData(json);
  } catch {
    return [];
  }
}

export function buildDetailUrlsFromSearchApiData(payload: unknown): string[] {
  const data = (payload as { data?: unknown }).data;
  if (!Array.isArray(data)) {
    return [];
  }

  return data
    .map((item) => {
      const stationId = (item as { station_id?: unknown }).station_id;
      const startTime = (item as { start_time?: unknown }).start_time;
      if (typeof stationId !== "string" || typeof startTime !== "string") {
        return null;
      }
      const ft = toFtTimestamp(startTime);
      if (!ft) {
        return null;
      }
      return `https://radiko.jp/#!/ts/${stationId}/${ft}`;
    })
    .filter((detailUrl): detailUrl is string => !!detailUrl);
}

function extractSearchKeyFromUrl(url: string): string | null {
  try {
    const parsed = new URL(url);
    const hash = parsed.hash.startsWith("#!")
      ? parsed.hash.slice(2)
      : parsed.hash;
    const queryIndex = hash.indexOf("?");
    if (queryIndex < 0) {
      return null;
    }
    const qs = new URLSearchParams(hash.slice(queryIndex + 1));
    const key = qs.get("key");
    return key && key.length > 0 ? key : null;
  } catch {
    return null;
  }
}

function toFtTimestamp(startTime: string): string | null {
  const m = startTime.match(
    /^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2}):(\d{2})$/,
  );
  if (!m) {
    return null;
  }
  return `${m[1]}${m[2]}${m[3]}${m[4]}${m[5]}${m[6]}`;
}

function randomUid(): string {
  return Array.from({ length: 32 }, () =>
    Math.floor(Math.random() * 16).toString(16),
  ).join("");
}
