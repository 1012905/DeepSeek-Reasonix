// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { SidePanel } from "./SidePanel";
import { MockPort } from "../port/mock";
import type { AgentPort } from "../port/port";

afterEach(() => {
  cleanup();
  window.getSelection()?.removeAllRanges();
});

function draw(extra: Partial<AgentPort> = {}) {
  const port = Object.assign(new MockPort(), extra) as unknown as AgentPort;
  const onClose = vi.fn();
  const view = render(<SidePanel port={port} onClose={onClose} />);
  return { port, onClose, view };
}

function select(text: string) {
  const host = document.createElement("p");
  host.textContent = text;
  document.body.appendChild(host);
  const range = document.createRange();
  range.selectNodeContents(host);
  const sel = window.getSelection()!;
  sel.removeAllRanges();
  sel.addRange(range);
  return host;
}

describe("the side conversation panel", () => {
  it("opens a conversation of its own and answers a question in it", async () => {
    const { port } = draw();
    const open = vi.spyOn(port, "openSide");
    const ask = vi.spyOn(port, "askSide");
    await userEvent.type(await screen.findByRole("textbox", { name: "向辅助会话提问" }), "这个函数做什么？");
    await userEvent.click(screen.getByRole("button", { name: "发送" }));
    await waitFor(() => expect(ask).toHaveBeenCalledOnce());
    expect(open.mock.calls.length).toBeLessThanOrEqual(1);
    expect(ask.mock.calls[0]![1]).toBe("这个函数做什么？");
    expect(ask.mock.calls[0]![2]).toEqual([]);
    expect(await screen.findByText(/只读回答/)).toBeTruthy();
    expect(screen.getByText("这个函数做什么？")).toBeTruthy();
  });

  it("brings the selected text of the main conversation along as a quote, once", async () => {
    const { port } = draw();
    const ask = vi.spyOn(port, "askSide");
    const quoted = select("第 3 行的 panic 是怎么来的");
    await userEvent.click(await screen.findByRole("button", { name: "引用选中内容" }));
    expect(within(document.querySelector(".sidequotes") as HTMLElement).getByText(/第 3 行的 panic/)).toBeTruthy();
    quoted.remove();
    await userEvent.type(screen.getByRole("textbox", { name: "向辅助会话提问" }), "解释一下");
    await userEvent.click(screen.getByRole("button", { name: "发送" }));
    await waitFor(() => expect(ask).toHaveBeenCalledWith(expect.any(String), "解释一下", ["第 3 行的 panic 是怎么来的"]));

    await userEvent.type(screen.getByRole("textbox", { name: "向辅助会话提问" }), "再问一个");
    await userEvent.click(screen.getByRole("button", { name: "发送" }));
    await waitFor(() => expect(ask).toHaveBeenCalledTimes(2));
    expect(ask.mock.calls[1]![2]).toEqual([]);
  });

  it("lets a quote be taken back before it is sent", async () => {
    const { port } = draw();
    const ask = vi.spyOn(port, "askSide");
    select("先引用再取消");
    await userEvent.click(await screen.findByRole("button", { name: "引用选中内容" }));
    await userEvent.click(screen.getByRole("button", { name: /移除引用/ }));
    await userEvent.type(screen.getByRole("textbox", { name: "向辅助会话提问" }), "问题");
    await userEvent.click(screen.getByRole("button", { name: "发送" }));
    await waitFor(() => expect(ask).toHaveBeenCalledWith(expect.any(String), "问题", []));
  });

  it("discards the conversation when it is closed", async () => {
    const { port, onClose } = draw();
    const close = vi.spyOn(port, "closeSide");
    await userEvent.type(await screen.findByRole("textbox", { name: "向辅助会话提问" }), "问");
    await userEvent.click(screen.getByRole("button", { name: "发送" }));
    await screen.findByText(/只读回答/);
    await userEvent.click(screen.getByRole("button", { name: "关闭辅助会话" }));
    expect(close).toHaveBeenCalledOnce();
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("closes the conversation it opened even when it is unmounted without a click", async () => {
    const { port, view } = draw();
    const close = vi.spyOn(port, "closeSide");
    await userEvent.type(await screen.findByRole("textbox", { name: "向辅助会话提问" }), "问");
    await userEvent.click(screen.getByRole("button", { name: "发送" }));
    await screen.findByText(/只读回答/);
    view.unmount();
    await waitFor(() => expect(close).toHaveBeenCalledOnce());
  });

  it("says so when the question fails, and keeps what was typed", async () => {
    const { port } = draw();
    vi.spyOn(port, "askSide").mockRejectedValue(new Error("模型没有响应"));
    const box = await screen.findByRole("textbox", { name: "向辅助会话提问" });
    await userEvent.type(box, "我的问题");
    await userEvent.click(screen.getByRole("button", { name: "发送" }));
    expect((await screen.findByRole("alert")).textContent).toContain("模型没有响应");
    expect((box as HTMLTextAreaElement).value).toBe("我的问题");
  });

  it("holds the send button while an answer is being written, and for an empty question", async () => {
    const { port } = draw();
    let release!: (v: string) => void;
    vi.spyOn(port, "askSide").mockImplementation(() => new Promise((r) => { release = r; }));
    const send = screen.getByRole("button", { name: "发送" }) as HTMLButtonElement;
    expect(send.disabled).toBe(true);
    await userEvent.type(await screen.findByRole("textbox", { name: "向辅助会话提问" }), "q");
    await userEvent.click(send);
    await waitFor(() => expect(send.disabled).toBe(true));
    release("done");
    await screen.findByText("done");
  });
});
