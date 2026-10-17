// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { MockPort } from "../port/mock";
import { useWorkspaceBranches } from "./useWorkspaceBranches";
import type { WorkspaceGit } from "../port/port";

afterEach(cleanup);

it("releases the switch state on port change and ignores the old completion", async () => {
  const old = new MockPort(), next = new MockPort();
  let resolve!: (value: WorkspaceGit) => void;
  vi.spyOn(old, "switchBranch").mockImplementation(() => new Promise((done) => { resolve = done; }));
  const changed = vi.fn(), switched = vi.fn(), error = vi.fn();
  const { result, rerender } = renderHook(({ port }) => useWorkspaceBranches(port, changed, switched, error), { initialProps: { port: old } });
  act(() => result.current.switchTo("studio"));
  expect(result.current.moving).toBe("studio");
  rerender({ port: next });
  expect(result.current.moving).toBeNull();
  await act(async () => { resolve(await next.workspaceGit()); });
  expect(changed).not.toHaveBeenCalled();
  expect(switched).not.toHaveBeenCalled();
  expect(error).not.toHaveBeenCalled();
});
