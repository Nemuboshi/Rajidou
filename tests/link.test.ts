import { describe, expect, it } from "vitest";
import {
  classifyRadikoLink,
  extractDetailFromDetailUrl,
  pickLatestDetailUrl,
} from "../src/core/link.js";

describe("classifyRadikoLink", () => {
  it("classifies search links", () => {
    const url = "https://radiko.jp/#!/search/timeshift?key=sora%20to%20hoshi";
    expect(classifyRadikoLink(url)).toBe("search");
  });

  it("classifies detail links", () => {
    const url = "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000";
    expect(classifyRadikoLink(url)).toBe("detail");
  });
});

describe("extractDetailFromDetailUrl", () => {
  it("extracts station and ft", () => {
    const url = "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000";
    expect(extractDetailFromDetailUrl(url)).toEqual({
      stationId: "ALPHA-STATION",
      ft: "20260219000000",
    });
  });
});

describe("pickLatestDetailUrl", () => {
  it("returns latest detail entry by ft", () => {
    const links = [
      "https://radiko.jp/#!/ts/ALPHA-STATION/20260217000000",
      "https://radiko.jp/#!/ts/ALPHA-STATION/20260218000000",
      "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000",
    ];
    expect(pickLatestDetailUrl(links)).toBe(
      "https://radiko.jp/#!/ts/ALPHA-STATION/20260219000000",
    );
  });
});
