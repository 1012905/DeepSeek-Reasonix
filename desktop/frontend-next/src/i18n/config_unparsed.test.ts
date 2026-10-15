// @vitest-environment jsdom
import { describe, expect, it } from "vitest";
import { HttpError } from "../port/http_error";
import { STORAGE, boot } from "./index";
import { reason } from "./kernel";

const refused = new HttpError(409, "config does not parse", { code: "config.unparsed", params: { path: "C:\\Users\\a\\.reasonix\\config.toml", line: 12 } });

describe("a settings save refused because the config file does not parse", () => {
  it("says where to see and repair it, not only that it failed", () => {
    expect(reason(refused)).toContain("设置");
  });

  it("says the same in English", () => {
    localStorage.setItem(STORAGE, "en");
    boot();
    try {
      expect(reason(refused)).toContain("Settings");
    } finally {
      localStorage.setItem(STORAGE, "zh");
      boot();
    }
  });
});
