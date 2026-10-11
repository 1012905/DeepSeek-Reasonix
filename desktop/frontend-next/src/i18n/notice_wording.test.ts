import { describe, expect, it } from "vitest";
import { NOTICE_TEXT } from "./notices";
import { NOTICE_VAR_CODES, hasTypedDetail, noticeWording } from "./notice_wording";

const placeholders = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]);

const PAYLOAD: Record<string, string | undefined> = {
  extension_skipped: JSON.stringify({ extension: "e", point: "tool.before" }),
  inbox_recovered: JSON.stringify({ count: 2 }),
  permission_rules_dormant: JSON.stringify({ rules: [{ list: "ask", rule: "Foo(bar)" }] }),
  job_finished: JSON.stringify({ id: "j", label: "build" }),
  job_killed: JSON.stringify({ id: "j", label: "build" }),
  job_failed: JSON.stringify({ id: "j", label: "build" }),
  display_currency: "USD",
  compact_declined: undefined,
  compact_failed: "summary_failed",
  compact_held: "summary_failed",
};

describe("notice wording", () => {
  const withSlots = Object.entries(NOTICE_TEXT).filter(([, s]) => placeholders(s).length > 0);

  it.each(withSlots)("%s registers a reader for every placeholder it has", (code, sentence) => {
    expect(NOTICE_VAR_CODES, `${code} has placeholders but no variable reader in notice_wording.ts`).toContain(code);
    expect(code in PAYLOAD, `add a sample payload for ${code}`).toBe(true);
    const w = noticeWording(code, PAYLOAD[code]);
    expect(w).toBeDefined();
    for (const key of placeholders(sentence)) expect(w?.vars).toHaveProperty(key);
  });

  it.each(NOTICE_VAR_CODES)("%s yields nothing, never a half-filled sentence, for an unreadable detail", (code) => {
    if (code === "display_currency" || code === "compact_declined") return;
    expect(noticeWording(code, "not json")).toBeUndefined();
  });

  it("words a placeholder-free code without variables", () => {
    expect(noticeWording("verification_stalled", "anything")?.vars).toBeUndefined();
    expect(hasTypedDetail("verification_stalled")).toBe(false);
  });

  it("has no wording for an unknown code", () => {
    expect(noticeWording("some_future_code", "x")).toBeUndefined();
    expect(noticeWording(undefined, "x")).toBeUndefined();
  });
});
