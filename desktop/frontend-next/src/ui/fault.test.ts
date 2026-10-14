import { afterEach, describe, expect, it, vi } from "vitest";
import { noteFault } from "./fault";

afterEach(() => vi.restoreAllMocks());

describe("noting a fault of the page", () => {
  it("writes the name, a cut message and the stack frames for a TypeError", () => {
    const log = vi.spyOn(console, "error").mockImplementation(() => {});
    const e = new TypeError("Cannot read properties of undefined (reading 'filter')");
    e.stack = "TypeError: Cannot read properties of undefined (reading 'filter')\n    at foo (app.js:1:2)\n    at bar (app.js:3:4)";
    noteFault(e);
    expect(log).toHaveBeenCalledOnce();
    const line = String(log.mock.calls[0][0]);
    expect(line).toContain("TypeError");
    expect(line).toContain("(reading 'filter')");
    expect(line).toContain("at foo (app.js:1:2)");
  });

  it("says nothing for a refusal or anything that is not a defect of the page", () => {
    const log = vi.spyOn(console, "error").mockImplementation(() => {});
    noteFault(new Error("model refused"));
    noteFault("plain text");
    noteFault(undefined);
    expect(log).not.toHaveBeenCalled();
  });

  it("cuts a long message and keeps the frames bounded", () => {
    const log = vi.spyOn(console, "error").mockImplementation(() => {});
    const e = new RangeError("x".repeat(5000));
    e.stack = ["RangeError: x", ...Array.from({ length: 50 }, (_, i) => `    at f${i} (a.js:${i}:1)`)].join("\n");
    noteFault(e);
    const line = String(log.mock.calls[0][0]);
    expect(line.length).toBeLessThan(1500);
    expect(line.match(/\n/g)?.length).toBe(12);
  });
});
