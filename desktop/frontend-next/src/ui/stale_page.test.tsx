// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { entryScript, useStalePage } from "./stale_page";

const page = (src: string) => `<!doctype html><html><head><script type="module" crossorigin src="${src}"></script></head><body></body></html>`;

function Probe({ enabled }: { enabled: boolean }) {
  return <div data-testid="stale">{useStalePage(enabled) ? "stale" : "fresh"}</div>;
}

beforeEach(() => {
  document.head.innerHTML = '<script type="module" src="/assets/index-AAA.js"></script>';
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("entryScript", () => {
  it("reads the module entry, whatever attributes surround it", () => {
    expect(entryScript(page("/assets/index-BBB.js"))).toBe("/assets/index-BBB.js");
    expect(entryScript("<html>no script</html>")).toBeNull();
  });
});

describe("a page left open across a deploy", () => {
  const answer = (html: string | Error) =>
    vi.stubGlobal("fetch", vi.fn(async () => {
      if (html instanceof Error) throw html;
      return { ok: true, text: async () => html };
    }));

  it("says so once the hosted entry is a different build, on return to the tab", async () => {
    answer(page("/assets/index-BBB.js"));
    render(<Probe enabled />);
    expect(screen.getByTestId("stale").textContent).toBe("fresh");
    document.dispatchEvent(new Event("visibilitychange"));
    await waitFor(() => expect(screen.getByTestId("stale").textContent).toBe("stale"));
  });

  it("stays quiet when the entry is the one running", async () => {
    answer(page("/assets/index-AAA.js"));
    render(<Probe enabled />);
    document.dispatchEvent(new Event("visibilitychange"));
    await waitFor(() => expect(fetch).toHaveBeenCalled());
    expect(screen.getByTestId("stale").textContent).toBe("fresh");
  });

  it("does not guess from a failed or unreadable answer", async () => {
    answer(new Error("offline"));
    render(<Probe enabled />);
    document.dispatchEvent(new Event("visibilitychange"));
    await waitFor(() => expect(fetch).toHaveBeenCalled());
    expect(screen.getByTestId("stale").textContent).toBe("fresh");
  });

  it("asks nothing when the page is not the hosted one", () => {
    answer(page("/assets/index-BBB.js"));
    render(<Probe enabled={false} />);
    document.dispatchEvent(new Event("visibilitychange"));
    expect(fetch).not.toHaveBeenCalled();
  });
});
