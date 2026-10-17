// @vitest-environment jsdom
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { MockPort } from "../port/mock";
import { useWorktreeSetup } from "./useWorktreeSetup";
import type { WorktreePreparation } from "../port/workspace";

afterEach(cleanup);

const selection = { sourceRoot: "/source", ref: "refs/heads/main" };
const state = (id: string, phase: string): WorktreePreparation => ({ id, phase, ...selection, startedAt: Date.now(), workspaceRoot: "/target", branch: "task" });

it("cancels an old port and never hands its completed workspace to the new pane", async () => {
  const old = new MockPort(), next = new MockPort();
  let resolve!: (value: WorktreePreparation) => void;
  const start = vi.spyOn(old, "prepareWorktree").mockImplementation(() => new Promise((done) => { resolve = done; }));
  const cancel = vi.spyOn(old, "cancelWorktreePreparation").mockImplementation(async (id) => state(id, "cancelled"));
  const changed = vi.fn();
  const { result, rerender } = renderHook(({ port }) => useWorktreeSetup(port, true, "/source", changed), { initialProps: { port: old } });
  await act(async () => { result.current.setSelection(selection); });
  let preparation!: Promise<boolean>;
  act(() => { preparation = result.current.prepare([]); });
  rerender({ port: next });
  await act(async () => { resolve(state(start.mock.calls[0][0], "ready")); expect(await preparation).toBe(false); });
  expect(cancel).toHaveBeenCalledWith(start.mock.calls[0][0]);
  expect(changed).not.toHaveBeenCalled();
  expect(result.current.stage).toBeNull();
});

it("keeps a published workspace but does not send when cancellation loses publication", async () => {
  const port = new MockPort();
  let resolve!: (value: WorktreePreparation) => void;
  const start = vi.spyOn(port, "prepareWorktree").mockImplementation(() => new Promise((done) => { resolve = done; }));
  vi.spyOn(port, "cancelWorktreePreparation").mockImplementation(async (id) => state(id, "ready"));
  const { result } = renderHook(() => useWorktreeSetup(port, true, "/source", vi.fn()));
  await act(async () => { result.current.setSelection(selection); });
  let preparation!: Promise<boolean>;
  act(() => { preparation = result.current.prepare([]); });
  await act(async () => { await result.current.cancel(); });
  await act(async () => { resolve(state(start.mock.calls[0][0], "ready")); expect(await preparation).toBe(false); });
  expect(result.current.stage?.phase).toBe("retained");
});

it("does not create a replacement when the first operation cannot be reconciled", async () => {
  const port = new MockPort();
  const start = vi.spyOn(port, "prepareWorktree").mockRejectedValue(new Error("lost acknowledgement"));
  const read = vi.spyOn(port, "worktreePreparation").mockRejectedValue(new Error("offline"));
  const { result } = renderHook(() => useWorktreeSetup(port, true, "/source", vi.fn()));
  await act(async () => { result.current.setSelection(selection); });
  await act(async () => { await expect(result.current.prepare([])).rejects.toThrow(); });
  await waitFor(() => expect(result.current.stage?.phase).toBe("unreachable"));
  await act(async () => { await expect(result.current.prepare([])).rejects.toThrow(); });
  expect(start).toHaveBeenCalledOnce();
  expect(read.mock.calls.every(([id]) => id === start.mock.calls[0][0])).toBe(true);
});
