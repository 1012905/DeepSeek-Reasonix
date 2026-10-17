// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { WorktreeSetupPage } from "./WorktreeSetupPage";
import type { WorktreePreparation } from "../port/workspace";

afterEach(() => { cleanup(); vi.useRealTimers(); });
const status = (phase: string): WorktreePreparation => ({ id: "operation", phase, ref: "refs/heads/main", sourceRoot: "/source/project", workspaceRoot: "/worktrees/project", branch: "reasonix/task", startedAt: Date.now() });
const draw = (phase: string, code?: string) => {
  const retry = vi.fn(), cancel = vi.fn(), edit = vi.fn();
  const view = render(<WorktreeSetupPage stage={{ ...status(phase), code, detail: "diagnostic" }} task="保留的任务" files={["input.txt"]} busy={false} onRetry={retry} onCancel={cancel} onEdit={edit}/>);
  return { ...view, retry, cancel, edit };
};

it("retains task and attachments during a long wait and lets the user cancel", () => {
  vi.useFakeTimers();
  const { cancel } = draw("checking_out");
  expect(screen.getByText("保留的任务")).toBeTruthy();
  expect(screen.getByText("input.txt")).toBeTruthy();
  act(() => vi.advanceTimersByTime(16000));
  expect(screen.getByRole("status").textContent).toContain("仍在等待");
  fireEvent.click(screen.getByRole("button", { name: "取消创建" }));
  expect(cancel).toHaveBeenCalledOnce();
});

it("keeps failure details behind disclosure and retries without losing the task", () => {
  const { retry } = draw("failed", "workspace.worktree_timeout");
  expect(screen.queryByText("diagnostic")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "查看详情" }));
  expect(screen.getByText("diagnostic")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "重试创建" }));
  expect(retry).toHaveBeenCalledOnce();
  expect(screen.getByText("保留的任务")).toBeTruthy();
});

it("does not offer another creation while cleanup is uncertain", () => {
  draw("cleanup_failed", "workspace.worktree_cleanup_failed");
  expect(screen.queryByRole("button", { name: "重试创建" })).toBeNull();
  const cancel = screen.queryByRole("button", { name: "取消创建" }) as HTMLButtonElement | null;
  expect(cancel === null || cancel.disabled).toBe(true);
});

it("restores editing after confirmed cancellation and exposes the retained destination", () => {
  const { edit } = draw("cancelled");
  fireEvent.click(screen.getByRole("button", { name: "返回编辑" }));
  expect(edit).toHaveBeenCalledOnce();
  expect(screen.getByText("input.txt")).toBeTruthy();
});

it("does not claim automatic submission when a late cancellation retained the workspace", () => {
  draw("retained");
  expect(screen.getByText("目录已经创建成功，取消未能撤回目录切换。任务尚未发送。")).toBeTruthy();
  expect(screen.queryByText("即将进入对话，自动开始任务。")).toBeNull();
});
