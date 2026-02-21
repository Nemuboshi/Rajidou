const SEARCH_MARKER = "#!/search/timeshift";
const DETAIL_PREFIX = "#!/ts/";

/**
 * Source map in this file:
 * - All functions are new CLI helpers, designed to parse/route Radiko hash URLs.
 */
export type RadikoLinkKind = "search" | "detail" | "unsupported";

export interface DetailRef {
  stationId: string;
  ft: string;
}

/**
 * Source: New helper for CLI URL routing (not directly extracted from rajiko).
 */
export function classifyRadikoLink(url: string): RadikoLinkKind {
  if (url.includes(SEARCH_MARKER)) {
    return "search";
  }
  if (url.includes(DETAIL_PREFIX)) {
    return "detail";
  }
  return "unsupported";
}

/**
 * Source: New helper for CLI detail URL parsing (compatible with rajiko URL format).
 */
export function extractDetailFromDetailUrl(url: string): DetailRef {
  const normalized = new URL(url);
  const hash = normalized.hash.startsWith("#!")
    ? normalized.hash.slice(2)
    : normalized.hash.slice(1);
  const segments = hash.split("/").filter(Boolean);
  if (segments.length < 3 || segments[0] !== "ts") {
    throw new Error(`Invalid detail URL: ${url}`);
  }

  const stationId = segments[1];
  const ft = segments[2];
  if (!/^\d{14}$/.test(ft)) {
    throw new Error(`Invalid ft timestamp in detail URL: ${url}`);
  }

  return { stationId, ft };
}

/**
 * Source: New helper for search-page result selection.
 */
export function pickLatestDetailUrl(
  urls: string[],
  now: Date = new Date(),
): string {
  const nowTs = toTimestamp(now);
  const details = urls
    .map((url) => ({ url, detail: extractDetailFromDetailUrl(url) }))
    .filter((x) => x.detail.ft <= nowTs)
    .sort((a, b) => b.detail.ft.localeCompare(a.detail.ft));

  if (details.length === 0) {
    throw new Error("No usable detail URL found in search results.");
  }
  return details[0].url;
}

function toTimestamp(value: Date): string {
  const y = value.getFullYear().toString().padStart(4, "0");
  const m = (value.getMonth() + 1).toString().padStart(2, "0");
  const d = value.getDate().toString().padStart(2, "0");
  const hh = value.getHours().toString().padStart(2, "0");
  const mm = value.getMinutes().toString().padStart(2, "0");
  const ss = value.getSeconds().toString().padStart(2, "0");
  return `${y}${m}${d}${hh}${mm}${ss}`;
}
