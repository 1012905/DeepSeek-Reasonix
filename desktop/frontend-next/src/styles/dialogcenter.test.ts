import { expect, it } from "vitest";

const CSS = import.meta.glob("./dialogcenter.css", { query: "?raw", import: "default", eager: true }) as Record<string, string>;
const css = CSS["./dialogcenter.css"]!;

it("keeps the transparent bottom border that centres a dialog under the title bar", () => {
  expect(css).toMatch(/:is\(\.prefs, \.fbk-veil, \.image-preview, \.cpv, \.agent-tx-scrim\)\s*\{\s*border-bottom:\s*var\(--chrome-h, 39px\) solid transparent;/);
});

it("takes it off a settings window that has been enlarged, so the sheet reaches the bottom of the window", () => {
  const rule = css.match(/\.prefs:has\(> \.prefs-sheet\[data-wide\]\)\s*\{([^}]*)\}/)?.[1] ?? "";
  expect(rule).toMatch(/border-bottom-width:\s*0/);
});
