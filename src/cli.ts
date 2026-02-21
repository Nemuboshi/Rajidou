import { readFile } from "node:fs/promises";
import path from "node:path";
import YAML from "yaml";
import { Logger } from "./cli/logger.js";
import { DownloadProgress } from "./cli/progress.js";
import { downloadFromDetailUrl } from "./core/downloader.js";
import { resolveToDetailUrl } from "./core/page-resolver.js";

interface ConfigFile {
  links: string[];
  outputDir?: string;
  areaId?: string;
}

function parseArgs(argv: string[]): { configPath: string } {
  const index = argv.findIndex((arg) => arg === "-c" || arg === "--config");
  if (index >= 0 && argv[index + 1]) {
    return { configPath: argv[index + 1] };
  }
  return { configPath: "links.yaml" };
}

async function loadConfig(filePath: string): Promise<ConfigFile> {
  const raw = await readFile(filePath, "utf-8");
  const parsed = YAML.parse(raw) as ConfigFile;
  if (!parsed || !Array.isArray(parsed.links) || parsed.links.length === 0) {
    throw new Error("Config must contain a non-empty `links` array.");
  }
  return parsed;
}

async function main(): Promise<void> {
  const logger = new Logger();
  const { configPath } = parseArgs(process.argv.slice(2));
  const resolvedConfigPath = path.resolve(configPath);
  const config = await loadConfig(resolvedConfigPath);
  const outputDir = path.resolve(config.outputDir ?? "downloads");
  const failures: Array<{ inputUrl: string; error: string }> = [];
  let successCount = 0;
  for (const inputUrl of config.links) {
    const progress = new DownloadProgress("segments");
    try {
      logger.info(`Input: ${inputUrl}`);
      const detailUrl = await resolveToDetailUrl(inputUrl);
      logger.info(`Resolved detail: ${detailUrl}`);

      const outputPath = await downloadFromDetailUrl(detailUrl, {
        outputDir,
        areaId: config.areaId,
        onProgress: (done, total) => progress.update(done, total),
      });
      progress.stop();
      logger.success(`Downloaded: ${outputPath}`);
      successCount += 1;
    } catch (error) {
      progress.stop();
      const message = formatError(error);
      failures.push({ inputUrl, error: message });
      logger.failure(`${inputUrl} -> ${message}`);
    }
  }

  logger.info(`Completed. success=${successCount} failed=${failures.length}`);
  if (failures.length > 0) {
    failures.forEach((item) => {
      logger.warn(`Failure detail: ${item.inputUrl} :: ${item.error}`);
    });
    process.exitCode = 2;
  }
}

main().catch((err) => {
  const logger = new Logger();
  logger.error(formatError(err));
  process.exitCode = 1;
});

function formatError(error: unknown): string {
  if (!(error instanceof Error)) {
    return String(error);
  }
  const parts: string[] = [error.message];
  let current: unknown = error;
  let guard = 0;
  while (current && guard < 4) {
    guard += 1;
    const cause = (current as { cause?: unknown }).cause;
    if (!cause) {
      break;
    }
    if (cause instanceof Error) {
      parts.push(`cause=${cause.message}`);
      const code = (cause as { code?: string }).code;
      if (code) {
        parts.push(`code=${code}`);
      }
      current = cause;
    } else {
      parts.push(`cause=${String(cause)}`);
      break;
    }
  }
  return parts.join(" | ");
}
