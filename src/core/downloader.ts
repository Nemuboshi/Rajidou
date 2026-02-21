import path from "node:path";
import { downloadAndMergeAacSegments } from "./audio.js";
import { retrieveToken } from "./auth.js";
import { buildProgramFileName } from "./filename.js";
import { extractDetailFromDetailUrl } from "./link.js";
import { buildSegmentUrls } from "./playlist.js";
import { resolveProgramMeta } from "./program.js";
import { resolveStationAreaId } from "./rajiko-derived.js";

/**
 * Source map in this file:
 * - downloadFromDetailUrl: rajiko/modules/timeshift.js (downloadtimeShift), adapted for Node CLI
 */
export interface DownloadOptions {
  outputDir: string;
  areaId?: string;
  onProgress?: (done: number, total: number) => void;
}

/**
 * Source: rajiko/modules/timeshift.js -> downloadtimeShift
 * Adaptation: Node workflow without browser extension storage/download APIs.
 */
export async function downloadFromDetailUrl(
  detailUrl: string,
  options: DownloadOptions,
): Promise<string> {
  const detail = extractDetailFromDetailUrl(detailUrl);
  const areaId =
    options.areaId ?? (await resolveStationAreaId(detail.stationId));
  const token = await retrieveToken(areaId);
  const meta = await resolveProgramMeta(detail.stationId, detail.ft);
  const segmentUrls = await buildSegmentUrls({
    stationId: detail.stationId,
    ft: meta.ft,
    to: meta.to,
    token,
    areaId,
  });

  const fileName = buildProgramFileName(meta.title, meta.ft);
  const outputPath = await downloadAndMergeAacSegments({
    segmentUrls,
    outputDir: options.outputDir,
    fileName,
    onProgress: options.onProgress,
  });
  return path.resolve(outputPath);
}
