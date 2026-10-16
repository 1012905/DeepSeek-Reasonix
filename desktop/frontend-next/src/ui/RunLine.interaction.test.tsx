// @vitest-environment jsdom
import "./testkit";
import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PaneShown } from "./shown";
import { RunLine } from "./RunLine";

const base = { label: "运行中", blocked: false, sent: 0, received: 0, estimated: false };

beforeEach(() => vi.useFakeTimers({ now: 1_000_000 }));
afterEach(() => { cleanup(); vi.useRealTimers(); });

describe("the running chip above the composer", () => {
  it("says how long the turn has been running and keeps counting", () => {
    render(<RunLine {...base} running since={1_000_000 - 65_000} />);
    expect(screen.getByText("1m 5s")).toBeTruthy();
    act(() => { vi.advanceTimersByTime(3_000); });
    expect(screen.getByText("1m 8s")).toBeTruthy();
  });

  it("shows no time for a turn this window did not see start", () => {
    const { container } = render(<RunLine {...base} running />);
    expect(container.querySelector(".studio-runclock")).toBeNull();
  });

  it("shows no time and runs no timer when nothing is running", () => {
    const { container } = render(<RunLine {...base} running={false} since={1_000_000 - 5_000} />);
    expect(container.querySelector(".studio-runclock")).toBeNull();
    expect(vi.getTimerCount()).toBe(0);
  });

  it("keeps the time out of what a screen reader announces", () => {
    const { container } = render(<RunLine {...base} running since={1_000_000 - 5_000} />);
    expect(container.querySelector(".studio-runclock")?.getAttribute("aria-hidden")).toBe("true");
  });

  it("does not tick in a pane nobody is looking at", () => {
    render(<PaneShown.Provider value={false}><RunLine {...base} running since={1_000_000 - 5_000} /></PaneShown.Provider>);
    expect(vi.getTimerCount()).toBe(0);
  });

  it("keeps the time while the turn waits on a decision", () => {
    render(<RunLine {...base} running blocked since={1_000_000 - 12_000} />);
    expect(screen.getByText("12s")).toBeTruthy();
  });
});
