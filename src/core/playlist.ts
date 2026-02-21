import { fetchWithRetry } from "./http.js";
import { parseTimestamp, stepTimestamp } from "./time.js";

/**
 * Source map in this file:
 * - playlistCreateUrl: rajiko/modules/timeshift.js (playlist_create_url)
 * - buildSegmentUrls: rajiko/modules/timeshift.js (downloadtimeShift seek/chunklist loop)
 */
/**
 * Source: rajiko/modules/timeshift.js -> playlist_create_url
 */
export async function playlistCreateUrl(stationId: string): Promise<string> {
  const resp = await fetchWithRetry(
    `https://radiko.jp/v3/station/stream/pc_html5/${stationId}.xml`,
    undefined,
    {
      retries: 3,
      delayMs: 300,
    },
  );
  if (!resp.ok) {
    throw new Error(`station stream xml failed: ${resp.status}`);
  }
  const xml = await resp.text();
  const regex =
    /((areafree="0".*?timefree="1")|(timefree="1".*?areafree="0")).*\n.*<playlist_create_url>(.*?)<\/playlist_create_url>/gm;
  const match = regex.exec(xml);
  if (match?.[4]) {
    return match[4];
  }
  return "https://tf-f-rpaa-radiko.smartstream.ne.jp/tf/playlist.m3u8";
}

/**
 * Source: rajiko/modules/timeshift.js -> downloadtimeShift (seek/chunklist logic)
 */
export async function buildSegmentUrls(input: {
  stationId: string;
  ft: string;
  to: string;
  token: string;
  areaId: string;
}): Promise<string[]> {
  const { stationId, ft, to, token, areaId } = input;
  const FIXED_SEEK = 300;
  const base = new URL(await playlistCreateUrl(stationId));
  base.searchParams.set("lsid", randomHex(32));
  base.searchParams.set("station_id", stationId);
  base.searchParams.set("l", String(FIXED_SEEK));
  base.searchParams.set("start_at", ft);
  base.searchParams.set("end_at", to);
  base.searchParams.set("type", "b");
  base.searchParams.set("ft", ft);
  base.searchParams.set("to", to);

  const links: string[] = [];
  let seek = ft;
  const endDate = parseTimestamp(to);
  while (parseTimestamp(seek) < endDate) {
    base.searchParams.set("seek", seek);
    const playlistResp = await fetchWithRetry(
      base.toString(),
      {
        headers: {
          "X-Radiko-AreaId": areaId,
          "X-Radiko-AuthToken": token,
        },
      },
      { retries: 3, delayMs: 250 },
    );
    const playlistText = await playlistResp.text();
    if (
      !playlistResp.ok ||
      playlistResp.status === 403 ||
      playlistText === "expired"
    ) {
      throw new Error(
        `playlist request failed at seek=${seek}: ${playlistResp.status}`,
      );
    }

    const detailUrl = firstDataLine(playlistText);
    const chunkResp = await fetchWithRetry(detailUrl, undefined, {
      retries: 3,
      delayMs: 250,
    });
    if (!chunkResp.ok) {
      throw new Error(`chunklist request failed: ${chunkResp.status}`);
    }
    const chunkText = await chunkResp.text();
    links.push(...allDataLines(chunkText));
    seek = stepTimestamp(seek, FIXED_SEEK);
  }

  return links;
}

function firstDataLine(m3u8: string): string {
  const line = allDataLines(m3u8)[0];
  if (!line) {
    throw new Error("m3u8 has no media lines");
  }
  return line;
}

function allDataLines(m3u8: string): string[] {
  return m3u8
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "" && !line.startsWith("#"));
}

function randomHex(length: number): string {
  const hex = "0123456789abcdef";
  let out = "";
  for (let i = 0; i < length; i += 1) {
    out += hex[Math.floor(Math.random() * hex.length)];
  }
  return out;
}
