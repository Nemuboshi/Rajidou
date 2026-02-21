import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { fetchWithRetry, retryOperation } from "./http.js";

/**
 * Source map in this file:
 * - parseAacPackedHeaderSize: rajiko/modules/util.js (parseAAC)
 * - runWithConcurrency: rajiko/modules/timeshift.js (worker)
 * - downloadAndMergeAacSegments: rajiko/modules/timeshift.js + rajiko/modules/recording.js
 */
/**
 * Source: rajiko/modules/util.js -> parseAAC
 */
export function parseAacPackedHeaderSize(data: Uint8Array): number {
  if (data.length < 10) {
    return 0;
  }
  if (data[0] !== 73 || data[1] !== 68 || data[2] !== 51) {
    return 0;
  }
  const id3PayloadSize = new DataView(
    data.buffer,
    data.byteOffset,
    data.byteLength,
  ).getUint32(6, false);
  return 10 + id3PayloadSize;
}

/**
 * Source: rajiko/modules/timeshift.js -> worker
 */
export async function runWithConcurrency<T, R>(
  items: T[],
  fn: (item: T, index: number) => Promise<R>,
  limit: number,
): Promise<R[]> {
  const results: R[] = [];
  let current = 0;

  async function worker(): Promise<void> {
    while (current < items.length) {
      const index = current;
      current += 1;
      results[index] = await fn(items[index], index);
    }
  }

  await Promise.all(
    Array.from({ length: Math.min(limit, items.length) }, () => worker()),
  );
  return results;
}

/**
 * Source: rajiko/modules/timeshift.js and rajiko/modules/recording.js
 * Adaptation: Node file writing replaces extension Blob/download APIs.
 */
export async function downloadAndMergeAacSegments(input: {
  segmentUrls: string[];
  outputDir: string;
  fileName: string;
  onProgress?: (done: number, total: number) => void;
}): Promise<string> {
  const { segmentUrls, outputDir, fileName, onProgress } = input;
  // Fixed worker count chosen for stability (fewer socket resets than aggressive parallelism).
  const concurrency = 4;
  await mkdir(outputDir, { recursive: true });
  let done = 0;
  const total = segmentUrls.length;
  onProgress?.(done, total);

  const segments = await runWithConcurrency(
    segmentUrls,
    async (url) => {
      const segment = await retryOperation(
        async () => {
          const resp = await fetchWithRetry(
            url,
            { credentials: "omit" },
            { retries: 1, delayMs: 150 },
          );
          if (!resp.ok) {
            throw new Error(`segment fetch failed: ${resp.status} ${url}`);
          }
          const buf = new Uint8Array(await resp.arrayBuffer());
          const headerSize = parseAacPackedHeaderSize(buf);
          return Buffer.from(buf.subarray(headerSize));
        },
        { retries: 3, delayMs: 300 },
      );
      done += 1;
      onProgress?.(done, total);
      return segment;
    },
    concurrency,
  );

  const final = Buffer.concat(segments);
  const outPath = path.resolve(outputDir, fileName);
  await writeFile(outPath, final);
  return outPath;
}
