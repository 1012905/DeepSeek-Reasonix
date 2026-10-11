// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { WorkspaceFolders } from "./WorkspaceFolders";
import { MockPort } from "../port/mock";
import { HttpError } from "../port/port";
import { boot, STORAGE, t } from "../i18n";
import type { AgentPort } from "../port/port";

beforeEach(() => {
  localStorage.setItem(STORAGE, "zh");
  boot();
});
afterEach(() => { cleanup(); vi.restoreAllMocks(); });

function draw(port = new MockPort() as unknown as AgentPort) {
  const onChanged = vi.fn();
  render(<WorkspaceFolders port={port} onChanged={onChanged} />);
  return { port, onChanged };
}

it("starts with no extra folder and says approval does not widen where writes can land", async () => {
  draw();
  expect(await screen.findByText(t("此项目的额外文件夹"))).toBeTruthy();
  expect(screen.queryAllByRole("button", { name: /移除文件夹/ })).toHaveLength(0);
  expect(screen.getByText(/「完全放行」只决定是否询问/)).toBeTruthy();
  expect((screen.getByRole("button", { name: t("添加") }) as HTMLButtonElement).disabled).toBe(true);
});

it("adds a folder, lists it, tells the window to rebuild, and removes it again", async () => {
  const { onChanged } = draw();
  const input = await screen.findByLabelText(t("要添加的文件夹路径"));
  await userEvent.type(input, "/Users/you/code/shared-lib");
  await userEvent.click(screen.getByRole("button", { name: t("添加") }));
  expect(await screen.findByText("/Users/you/code/shared-lib")).toBeTruthy();
  expect(onChanged).toHaveBeenCalledTimes(1);
  expect((input as HTMLInputElement).value).toBe("");
  await userEvent.click(screen.getByRole("button", { name: t("移除文件夹 {path}", { path: "/Users/you/code/shared-lib" }) }));
  await waitFor(() => expect(screen.queryByText("/Users/you/code/shared-lib")).toBeNull());
  expect(onChanged).toHaveBeenCalledTimes(2);
});

it.each([
  ["relative/path", "需要一个已存在的文件夹的绝对路径"],
  ["/Users/you/code/site/sub", "这个文件夹已经在当前工作区里"],
  ["/Users/you", "这个文件夹范围太大"],
  ["/Users/you/.ssh/keys", "这个文件夹含有或位于密钥"],
])("explains a refused folder %s in words and keeps what was typed", async (path, words) => {
  const { onChanged } = draw();
  const input = await screen.findByLabelText(t("要添加的文件夹路径"));
  await userEvent.type(input, path);
  await userEvent.click(screen.getByRole("button", { name: t("添加") }));
  expect((await screen.findByRole("alert")).textContent).toContain(words);
  expect((input as HTMLInputElement).value).toBe(path);
  expect(onChanged).not.toHaveBeenCalled();
});

it("says so when the kernel is busy, and when other panes will take the change later", async () => {
  const port = new MockPort() as unknown as AgentPort;
  vi.spyOn(port, "addWorkspaceRoot").mockRejectedValueOnce(new HttpError(409, "busy", { code: "workspace_roots.busy" }))
    .mockResolvedValueOnce({ primary: "/Users/you/code/site", extra: ["/x/y"], pending: 2 });
  draw(port);
  const input = await screen.findByLabelText(t("要添加的文件夹路径"));
  await userEvent.type(input, "/x/y");
  await userEvent.click(screen.getByRole("button", { name: t("添加") }));
  expect((await screen.findByRole("alert")).textContent).toContain("当前还有任务在运行");
  await userEvent.click(screen.getByRole("button", { name: t("添加") }));
  expect(await screen.findByText(/还有 2 个窗格正在运行/)).toBeTruthy();
  expect(screen.queryByRole("alert")).toBeNull();
});

it("adds the folder chosen in the system picker", async () => {
  const port = new MockPort() as unknown as AgentPort;
  vi.spyOn(port, "pickFolder").mockResolvedValue("/Users/you/code/picked");
  draw(port);
  await userEvent.click(await screen.findByRole("button", { name: t("选文件夹") }));
  expect(await screen.findByText("/Users/you/code/picked")).toBeTruthy();
});
