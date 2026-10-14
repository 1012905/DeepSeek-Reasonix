// @vitest-environment jsdom
import { act, cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import "./testkit";
import { Workspaces } from "./Workspaces";
import { SessionQuickSection } from "./SessionQuickSection";
import { setShowsSessionQuickActions, showsSessionQuickActions } from "../state/sessionQuick";
import type { TreeSession, TreeWorkspace } from "../port/hub";

beforeEach(() => setShowsSessionQuickActions(false));
afterEach(() => {
  cleanup();
  setShowsSessionQuickActions(false);
});

const PATH = "/w/.reasonix/sessions/session.jsonl";
const TITLE = "review the changes";

function draw(over: Partial<TreeSession> = {}, live = false) {
  const session = { path: PATH, name: "session", title: TITLE, ...over };
  const tree: TreeWorkspace[] = [{ root: "/w", name: "w", sessions: [session] }];
  const onArchive = vi.fn().mockResolvedValue(undefined);
  const onOpen = vi.fn().mockResolvedValue(undefined);
  render(<Workspaces
    hub={{ removeSession: vi.fn().mockResolvedValue(undefined) } as never} tree={tree} treeRead
    runtimes={session.runtimeId ? [{ id: session.runtimeId, base: "", root: "/w", name: "w", sessionPath: PATH }] : []}
    active="" folded={new Set()} onFold={() => {}} reload={vi.fn().mockResolvedValue(undefined)}
    onOpen={onOpen} onFocus={vi.fn()} onClose={vi.fn().mockResolvedValue(undefined)}
    liveIds={(ids) => live ? ids : []} runs={{}} scope="all"
    onArchive={onArchive} onRename={() => {}} onError={vi.fn()}
    adder={{ add: () => {}, close: () => {}, at: null } as never}
  />);
  return { onArchive, onOpen };
}

const row = () => screen.getByRole("treeitem", { name: new RegExp(TITLE) });
const buttons = () => within(row()).queryAllByRole("button", { name: /归档|删除/ });

it("draws no archive or delete button on a row until the setting is on", () => {
  draw();
  expect(buttons()).toEqual([]);
});

it("draws both buttons when the setting is on, and archives without opening the session", async () => {
  setShowsSessionQuickActions(true);
  const { onArchive, onOpen } = draw();
  expect(buttons().map((b) => b.getAttribute("data-action"))).toEqual(["session.archive", "session.delete"]);
  await userEvent.click(within(row()).getByRole("button", { name: `归档会话：${TITLE}` }));
  expect(onArchive).toHaveBeenCalledExactlyOnceWith(PATH, true, undefined);
  expect(onOpen).not.toHaveBeenCalled();
});

it("opens the delete confirmation from the row button and keeps archive off while the session runs", async () => {
  setShowsSessionQuickActions(true);
  draw({ runtimeId: "r1" }, true);
  expect((within(row()).getByRole("button", { name: `归档会话：${TITLE}` }) as HTMLButtonElement).disabled).toBe(true);
  await userEvent.click(within(row()).getByRole("button", { name: `删除会话：${TITLE}` }));
  expect(screen.getByRole("alertdialog", { name: `删除「${TITLE}」？` })).toBeTruthy();
});

it("shows or hides the buttons on a list that is already drawn when the setting flips", () => {
  draw();
  expect(buttons()).toEqual([]);
  act(() => setShowsSessionQuickActions(true));
  expect(buttons()).toHaveLength(2);
  act(() => setShowsSessionQuickActions(false));
  expect(buttons()).toEqual([]);
});

it("remembers the choice made in Settings and starts off", async () => {
  expect(showsSessionQuickActions()).toBe(false);
  render(<SessionQuickSection />);
  const show = screen.getByRole("button", { name: "显示" });
  expect(show.getAttribute("aria-pressed")).toBe("false");
  await userEvent.click(show);
  expect(showsSessionQuickActions()).toBe(true);
  expect(show.getAttribute("aria-pressed")).toBe("true");
  await userEvent.click(screen.getByRole("button", { name: "关闭" }));
  expect(showsSessionQuickActions()).toBe(false);
});
