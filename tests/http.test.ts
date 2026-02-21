import { describe, expect, it, vi } from "vitest";
import { retryOperation } from "../src/core/http.js";

describe("retryOperation", () => {
  it("retries transient failures and eventually succeeds", async () => {
    let attempts = 0;
    const result = await retryOperation(
      async () => {
        attempts += 1;
        if (attempts < 3) {
          throw new TypeError("fetch failed");
        }
        return "ok";
      },
      { retries: 3, delayMs: 1 },
    );
    expect(result).toBe("ok");
    expect(attempts).toBe(3);
  });

  it("throws after max retries", async () => {
    const fn = vi.fn(async () => {
      throw new TypeError("fetch failed");
    });
    await expect(
      retryOperation(fn, { retries: 2, delayMs: 1 }),
    ).rejects.toThrow("fetch failed");
    expect(fn).toHaveBeenCalledTimes(3);
  });
});
