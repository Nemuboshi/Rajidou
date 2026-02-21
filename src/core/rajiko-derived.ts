import { readFile } from "node:fs/promises";
import path from "node:path";
import { fetchWithRetry } from "./http.js";

/**
 * Source map in this file:
 * - loadRajikoAppKey: rajiko/modules/static.js (aSmartPhone8_fullkey_b64, APP_VERSION_MAP usage)
 * - genGps: rajiko/modules/util.js (genGPS)
 * - genDeviceInfo: rajiko/modules/util.js (genRandomInfo), adapted
 * - resolveStationAreaId: rajiko/modules/constants.js (radioAreaId behavior), adapted
 */
const STATIC_FILE = path.resolve("rajiko/modules/static.js");
let keyMaterialCache:
  | {
      appVersion: string;
      appId: string;
      appKeyBase64: string;
    }
  | undefined;

/**
 * Source: rajiko/modules/static.js
 * Original constants:
 *   - aSmartPhone8_fullkey_b64
 *   - APP_VERSION_MAP
 */
export async function loadRajikoAppKey(): Promise<{
  appVersion: string;
  appId: string;
  appKeyBase64: string;
}> {
  if (keyMaterialCache) {
    return keyMaterialCache;
  }
  const content = await readFile(STATIC_FILE, "utf-8");
  const keyMatch = content.match(
    /const aSmartPhone8_fullkey_b64 = "([\s\S]*?)";/,
  );
  if (!keyMatch) {
    throw new Error(`Cannot locate aSmartPhone8_fullkey_b64 in ${STATIC_FILE}`);
  }
  keyMaterialCache = {
    appVersion: "8.2.4",
    appId: "aSmartPhone8",
    appKeyBase64: keyMatch[1],
  };
  return keyMaterialCache;
}

// Coordinates in JP1..JP47 order.
// Source: rajiko/modules/static.js -> coordinates object values.
const AREA_COORDINATES: Array<[number, number]> = [
  [43.064615, 141.346807],
  [40.824308, 140.739998],
  [39.703619, 141.152684],
  [38.268837, 140.8721],
  [39.718614, 140.102364],
  [38.240436, 140.363633],
  [37.750299, 140.467551],
  [36.341811, 140.446793],
  [36.565725, 139.883565],
  [36.390668, 139.060406],
  [35.856999, 139.648849],
  [35.605057, 140.123306],
  [35.689488, 139.691706],
  [35.447507, 139.642345],
  [37.902552, 139.023095],
  [36.695291, 137.211338],
  [36.594682, 136.625573],
  [36.065178, 136.221527],
  [35.664158, 138.568449],
  [36.651299, 138.180956],
  [35.391227, 136.722291],
  [34.97712, 138.383084],
  [35.180188, 136.906565],
  [34.730283, 136.508588],
  [35.004531, 135.86859],
  [35.021247, 135.755597],
  [34.686297, 135.519661],
  [34.691269, 135.183071],
  [34.685334, 135.832742],
  [34.225987, 135.167509],
  [35.503891, 134.237736],
  [35.472295, 133.0505],
  [34.661751, 133.934406],
  [34.39656, 132.459622],
  [34.185956, 131.470649],
  [34.065718, 134.55936],
  [34.340149, 134.043444],
  [33.841624, 132.765681],
  [33.559706, 133.531079],
  [33.606576, 130.418297],
  [33.249442, 130.299794],
  [32.744839, 129.873756],
  [32.789827, 130.741667],
  [33.238172, 131.612619],
  [31.911096, 131.423893],
  [31.560146, 130.557978],
  [26.2124, 127.680932],
];

/**
 * Source: rajiko/modules/util.js -> genGPS
 */
export function genGps(areaId: string): string {
  const index = Number(areaId.replace(/^JP/, "")) - 1;
  const pos = AREA_COORDINATES[index];
  if (!pos) {
    throw new Error(`Invalid area id: ${areaId}`);
  }
  let [lat, lon] = pos;
  lat += (Math.random() / 40.0) * (Math.random() > 0.5 ? 1 : -1);
  lon += (Math.random() / 40.0) * (Math.random() > 0.5 ? 1 : -1);
  return `${lat.toFixed(6)},${lon.toFixed(6)},gps`;
}

/**
 * Source: rajiko/modules/util.js -> genRandomInfo (adapted to deterministic app version/app id)
 */
export function genDeviceInfo(appVersion: string): {
  appVersion: string;
  userId: string;
  userAgent: string;
  device: string;
} {
  const model = "Google Pixel 6";
  const sdk = "34";
  const device = `${sdk}.GQML3`;
  const userAgent = `Dalvik/2.1.0 (Linux; U; Android 14.0.0; ${model}/AP2A.240805.005.S4)`;
  const hex = "0123456789abcdef";
  let userId = "";
  for (let i = 0; i < 32; i += 1) {
    userId += hex[Math.floor(Math.random() * hex.length)];
  }
  return { appVersion, userId, userAgent, device };
}

const stationAreaCache = new Map<string, string>();

/**
 * Source: rajiko/modules/constants.js -> radioAreaId lookup behavior
 * Adaptation: build mapping dynamically by scanning official station list APIs.
 */
export async function resolveStationAreaId(stationId: string): Promise<string> {
  const cached = stationAreaCache.get(stationId);
  if (cached) {
    return cached;
  }

  for (let n = 1; n <= 47; n += 1) {
    const areaId = `JP${n}`;
    const resp = await fetchWithRetry(
      `https://radiko.jp/v3/station/list/${areaId}.xml`,
      undefined,
      {
        retries: 3,
        delayMs: 200,
      },
    );
    if (!resp.ok) {
      continue;
    }
    const xml = await resp.text();
    if (xml.includes(`<id>${stationId}</id>`)) {
      stationAreaCache.set(stationId, areaId);
      return areaId;
    }
  }
  throw new Error(`Cannot resolve area id for station: ${stationId}`);
}
