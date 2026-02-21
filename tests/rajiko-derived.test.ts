import { describe, expect, it } from "vitest";
import {
  RADIKO_APP_ID,
  RADIKO_APP_KEY_BASE64,
  RADIKO_APP_VERSION,
} from "../src/core/radiko-app-key.js";
import { loadRajikoAppKey } from "../src/core/rajiko-derived.js";

describe("loadRajikoAppKey", () => {
  it("loads embedded key material constants", async () => {
    const key = await loadRajikoAppKey();
    expect(key).toEqual({
      appVersion: RADIKO_APP_VERSION,
      appId: RADIKO_APP_ID,
      appKeyBase64: RADIKO_APP_KEY_BASE64,
    });
    expect(key.appKeyBase64.length).toBeGreaterThan(1000);
  });
});
