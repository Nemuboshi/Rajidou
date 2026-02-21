import { describe, expect, it } from "vitest";
import { buildProgramFileName } from "../src/core/filename.js";

describe("buildProgramFileName", () => {
  it("builds title and date based filename", () => {
    expect(
      buildProgramFileName("SORA to HOSHI no ORCHESTRA", "20260211230000"),
    ).toBe("SORA to HOSHI no ORCHESTRA - 20260211.aac");
  });

  it("sanitizes forbidden path chars", () => {
    expect(buildProgramFileName('A/B:C*D?"E<F>G|', "20260211230000")).toBe(
      "A_B_C_D_E_F_G_ - 20260211.aac",
    );
  });
});
