import type { Page } from "playwright";
import { classifyRadikoLink, pickLatestDetailUrl } from "./link.js";

/**
 * Source map in this file:
 * - resolveToDetailUrl: inspired by rajiko/pages/popup.js timeshift URL detection, adapted for Playwright scraping
 */
/**
 * Source inspiration: rajiko/pages/popup.js (timeshift link detection from page state)
 * Adaptation: resolve latest detail URL by scraping search page links.
 */
export async function resolveToDetailUrl(
  page: Page,
  url: string,
): Promise<string> {
  const kind = classifyRadikoLink(url);
  if (kind === "detail") {
    await page.goto(url, { waitUntil: "domcontentloaded", timeout: 120_000 });
    return url;
  }
  if (kind !== "search") {
    throw new Error(`Unsupported link: ${url}`);
  }

  await page.goto(url, { waitUntil: "domcontentloaded", timeout: 120_000 });
  await page.waitForTimeout(4000);
  const hrefs = await page.$$eval('a[href*="#!/ts/"]', (els) =>
    els
      .map((e) => e.getAttribute("href"))
      .filter((v): v is string => !!v)
      .map((href) => new URL(href, location.origin).toString()),
  );
  if (hrefs.length === 0) {
    throw new Error(`No detail links found in search page: ${url}`);
  }
  return pickLatestDetailUrl(hrefs);
}
