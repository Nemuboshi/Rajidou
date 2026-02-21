import { afterEach, describe, expect, it, vi } from "vitest";
import {
  buildDetailUrlsFromSearchApiData,
  resolveToDetailUrl,
} from "../src/core/page-resolver.js";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("resolveToDetailUrl", () => {
  it("resolves search URL via search API", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          data: [
            {
              station_id: "BAYFM78",
              start_time: "2026-02-21 01:30:00",
            },
          ],
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );

    const result = await resolveToDetailUrl(
      "https://radiko.jp/#!/search/timeshift?key=%E3%83%93%E3%82%BF%E3%83%9F%E3%83%B3M",
    );
    expect(result).toBe("https://radiko.jp/#!/ts/BAYFM78/20260221013000");
  });

  it("keeps detail URL unchanged", async () => {
    const input = "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000";
    const result = await resolveToDetailUrl(input);
    expect(result).toBe(input);
  });
});

describe("buildDetailUrlsFromSearchApiData", () => {
  it("converts search API data to detail URLs", () => {
    const links = buildDetailUrlsFromSearchApiData({
      data: [
        {
          station_id: "ALPHA-STATION",
          start_time: "2026-02-19 00:00:00",
        },
      ],
    });
    expect(links).toEqual([
      "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000",
    ]);
  });
});
