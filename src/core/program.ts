import { fetchWithRetry } from "./http.js";

/**
 * Source map in this file:
 * - resolveProgramMeta: behavior adapted from rajiko/modules/timeshift.js with title extraction
 */

/**
 * Source: rajiko/modules/timeshift.js comments and existing timeshift URL usage
 * Adaptation: use weekly XML API to resolve `to` and `title` for a detail page `ft`.
 */
export async function resolveProgramMeta(
  stationId: string,
  ft: string,
): Promise<{ ft: string; to: string; title: string }> {
  const url = `https://api.radiko.jp/program/v3/weekly/${stationId}.xml`;
  const resp = await fetchWithRetry(url, undefined, {
    retries: 3,
    delayMs: 300,
  });
  if (!resp.ok) {
    throw new Error(`weekly program xml failed: ${resp.status}`);
  }
  const xml = await resp.text();
  const escapedFt = escapeRegExp(ft);
  const exact = new RegExp(
    `<prog\\s+[^>]*ft="${escapedFt}"\\s+to="(\\d{14})"[^>]*>([\\s\\S]*?)<\\/prog>`,
    "g",
  );
  const matched = exact.exec(xml);
  if (matched?.[1] && matched?.[2]) {
    const block = matched[2];
    const titleMatch = block.match(/<title>([\s\S]*?)<\/title>/);
    const title = decodeXml((titleMatch?.[1] ?? "").trim());
    return { ft, to: matched[1], title };
  }
  throw new Error(
    `Cannot find program range for station=${stationId} ft=${ft}`,
  );
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function decodeXml(text: string): string {
  return text
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'");
}
