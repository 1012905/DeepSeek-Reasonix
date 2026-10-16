// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { HistoryHit, TreeWorkspace } from "../port/hub";
import { ContentMatches } from "./ContentMatches";
import { RailSearch } from "./railsearch";

const tree = [{ root: "/w/a", name: "a", sessions: [{ path: "/w/a/s1.jsonl", name: "s1", title: "Rotate keys" }] }] as unknown as TreeWorkspace[];
const hit: HistoryHit = { path: "/w/a/s1.jsonl", id: "s1", snippet: "rotate the zebra credentials", role: "user", messageIndex: 0, score: 1 };

function setup(search: (q: string) => Promise<HistoryHit[]>) {
  const onOpen = vi.fn(() => Promise.resolve());
  render(
    <RailSearch>
      <ContentMatches hub={{ searchHistory: search }} tree={tree} onOpen={onOpen} onError={() => {}} />
    </RailSearch>,
  );
  return onOpen;
}

describe("ContentMatches", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("asks only after the reader pauses on two or more characters, then opens the match in its workspace", async () => {
    const search = vi.fn(() => Promise.resolve([hit]));
    const onOpen = setup(search);
    const box = screen.getByRole("searchbox");

    fireEvent.change(box, { target: { value: "z" } });
    await act(async () => void vi.advanceTimersByTime(1000));
    expect(search).not.toHaveBeenCalled();

    fireEvent.change(box, { target: { value: "zebra" } });
    await act(async () => void vi.advanceTimersByTime(100));
    expect(search).not.toHaveBeenCalled();
    await act(async () => void vi.advanceTimersByTime(300));
    expect(search).toHaveBeenCalledWith("zebra");

    fireEvent.click(document.querySelector("[data-action='session.open-match']")!);
    expect(screen.getByText("Rotate keys")).toBeTruthy();
    expect(onOpen).toHaveBeenCalledWith({ root: "/w/a", sessionPath: "/w/a/s1.jsonl" });
  });

  it("says so when the index cannot answer instead of showing nothing", async () => {
    setup(() => Promise.reject(new Error("down")));
    fireEvent.change(screen.getByRole("searchbox"), { target: { value: "zebra" } });
    await act(async () => void vi.advanceTimersByTime(400));
    expect(document.querySelector("[data-state='failed']")).not.toBeNull();
  });
});
