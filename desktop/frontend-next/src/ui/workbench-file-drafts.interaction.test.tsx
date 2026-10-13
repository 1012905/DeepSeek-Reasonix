// @vitest-environment jsdom
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { EditorView } from "@codemirror/view";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import type { WorkspaceFile } from "../port/port";
import { MockPort } from "../port/mock";
import { WorkbenchPanel } from "./WorkbenchPanel";
import "./testkit";

Range.prototype.getClientRects = () => [] as unknown as DOMRectList;
Range.prototype.getBoundingClientRect = () => new DOMRect(0, 0, 0, 0);
afterEach(cleanup);

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
const file = (path: string, content = `saved ${path}`, revision = "r1") => ({ path, content, revision });
const button = (action: string, target?: string) => document.querySelector<HTMLButtonElement>(
  `[data-action="workbench.${action}"]${target ? `[data-target="${target}"]` : ""}`,
)!;
async function editor(value?: string) {
  return waitFor(() => {
    const element = document.querySelector(".cm-content");
    if (!element) throw new Error("editor not mounted");
    const view = EditorView.findFromDOM(element as HTMLElement)!;
    if (value !== undefined) expect(view.state.doc.toString()).toBe(value);
    return view;
  });
}
async function append(text: string) {
  const view = await editor();
  act(() => view.dispatch({ changes: { from: view.state.doc.length, insert: text } }));
}
function fixture() {
  const user = userEvent.setup();
  const port = new MockPort();
  vi.spyOn(port, "workspaceFiles").mockResolvedValue({ files: ["a.txt", "b.txt"], directories: [] });
  const read = vi.spyOn(port, "workspaceFile").mockImplementation(async (path) => file(path));
  const save = vi.spyOn(port, "saveWorkspaceFile").mockImplementation(async (file) => ({ ...file, revision: "r2" }));
  const props = { port, tabs: [], manual: false, shown: true, scheme: "light" as const, changes: [], workspaceRoot: "/one", onCloseManual: vi.fn(), onSurfaces: vi.fn(), onExternal: vi.fn() };
  const view = render(<WorkbenchPanel {...props} />);
  const open = async (path: string) => {
    await user.click(button("files"));
    await user.click(await screen.findByRole("button", { name: path }));
  };
  return { ...view, user, port, read, save, props, open };
}

it("preserves both dirty file tabs without saving or reloading either draft", async () => {
  const f = fixture();
  await f.open("a.txt");
  await editor("saved a.txt");
  await append(" UNSAVED USER WORK");
  await f.open("b.txt");
  await editor("saved b.txt");
  await append(" B draft");
  await f.user.click(button("tab", "file:a.txt"));
  await editor("saved a.txt UNSAVED USER WORK");
  await f.user.click(button("tab", "file:b.txt"));
  await editor("saved b.txt B draft");
  expect(f.save).not.toHaveBeenCalled();
  expect(f.read).toHaveBeenCalledTimes(2);
});

it("keeps a draft through launcher cancellation, hiding and closing then reopening its tab", async () => {
  const f = fixture();
  await f.open("a.txt");
  await append(" draft");
  await f.user.click(button("new"));
  await f.user.keyboard("{Escape}");
  await editor("saved a.txt draft");
  f.rerender(<WorkbenchPanel {...f.props} shown={false} />);
  f.rerender(<WorkbenchPanel {...f.props} />);
  await editor("saved a.txt draft");
  await f.user.click(button("close", "file:a.txt"));
  await f.open("b.txt");
  await editor("saved b.txt");
  await f.open("a.txt");
  await editor("saved a.txt draft");
  expect(f.save).not.toHaveBeenCalled();
});

it.each([false, true])("keeps a late save result on its own file (failure: %s)", async (fails) => {
  const f = fixture();
  const pending = deferred<WorkspaceFile>();
  f.save.mockReturnValueOnce(pending.promise);
  await f.open("a.txt");
  await append(" draft");
  await f.user.click(button("save"));
  await f.open("b.txt");
  await editor("saved b.txt");
  await append(" other");
  await act(async () => {
    if (fails) pending.reject(new Error("save refused"));
    else {
      f.read.mockImplementation(async (path) => path === "a.txt" ? file(path, "saved a.txt draft", "r2") : file(path));
      pending.resolve(file("a.txt", "saved a.txt draft", "r2"));
    }
  });
  await editor("saved b.txt other");
  expect(screen.queryByText("save refused")).toBeNull();
  await f.user.click(button("tab", "file:a.txt"));
  await editor("saved a.txt draft");
  expect(button("save").disabled).toBe(!fails);
  if (fails) expect(screen.getByRole("alert").textContent).toBe("save refused");
  expect(f.save.mock.calls[0][0]).toEqual(file("a.txt", "saved a.txt draft"));
});

it("retains edits typed during saving and suppresses repeated save shortcuts", async () => {
  const f = fixture();
  const pending = deferred<WorkspaceFile>();
  f.save.mockReturnValueOnce(pending.promise);
  await f.open("a.txt");
  await append(" draft");
  await f.user.click(button("save"));
  await append(" newer");
  (await editor()).focus();
  await f.user.keyboard("{Control>}s{/Control}");
  expect(f.save).toHaveBeenCalledTimes(1);
  await act(async () => pending.resolve(file("a.txt", "saved a.txt draft", "r2")));
  await editor("saved a.txt draft newer");
  expect(button("save").disabled).toBe(false);
  await f.user.click(button("save"));
  expect(f.save.mock.calls[1][0]).toEqual(file("a.txt", "saved a.txt draft newer", "r2"));
});
