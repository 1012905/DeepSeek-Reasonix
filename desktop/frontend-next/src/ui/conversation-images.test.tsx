// @vitest-environment jsdom
import { cleanup, render, renderHook } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import type { AgentPort } from "../port/port";
import { Markdown } from "./Markdown";
import { useConversationFiles } from "./useConversationFiles";

afterEach(cleanup);

const port = { workspaceImageURL: (path: string) => `/rt/r1/workspace/image?path=${encodeURIComponent(path)}`, revealsFiles: () => false, opensFiles: () => false } as unknown as AgentPort;
const refs = (root: string) => renderHook(() => useConversationFiles(port, root, false, true, () => {}, () => {})).result.current.refs;

it.each([
  ["out/chart.png", "/rt/r1/workspace/image?path=out%2Fchart.png"],
  ["./out/chart.png", "/rt/r1/workspace/image?path=out%2Fchart.png"],
  ["/work/out/chart.png", "/rt/r1/workspace/image?path=out%2Fchart.png"],
  ["file:///work/out/chart.png", "/rt/r1/workspace/image?path=out%2Fchart.png"],
  ["../secret.png", null],
  ["/etc/hosts.png", null],
  ["https://example.com/x.png", null],
])("loads the picture %s from the workspace's controlled channel, or leaves it alone", (src, expected) => {
  expect(refs("/work").image(src)).toBe(expected);
});

it("loads a Windows drive picture only from inside the workspace", () => {
  const at = refs("C:\\Work");
  expect(at.image("file:///C:/Work/out/a.png")).toBe("/rt/r1/workspace/image?path=out%2Fa.png");
  expect(at.image("C:\\Work\\out\\a.png")).toBe("/rt/r1/workspace/image?path=out%2Fa.png");
  expect(at.image("file:///C:/Users/me/Pictures/a.jpg")).toBeNull();
  expect(at.image("C:\\Users\\me\\Pictures\\a.jpg")).toBeNull();
});

it("draws a local picture outside the workspace as a note, not a broken image", () => {
  const at = refs("C:\\Work");
  const text = "![inside](file:///C:/Work/out/a.png)\n\n![outside](file:///C:/Users/me/Pictures/a.jpg)\n\n![bare](C:/Users/me/Pictures/b.jpg)\n\n![remote](https://example.com/c.png)";
  const { container } = render(<Markdown text={text} local={at} />);
  const imgs = [...container.querySelectorAll("img")].map((i) => i.getAttribute("src"));
  expect(imgs).toEqual(["/rt/r1/workspace/image?path=out%2Fa.png", "https://example.com/c.png"]);
  const notes = [...container.querySelectorAll(".md-outside-image")];
  expect(notes.map((n) => n.textContent)).toEqual(["outside", "bare"]);
  expect(notes.every((n) => (n.getAttribute("title") ?? "").length > 0)).toBe(true);
});

it("leaves a picture alone where no conversation owns the workspace", () => {
  const { container } = render(<Markdown text="![x](file:///C:/Users/me/a.jpg)" />);
  expect(container.querySelector(".md-outside-image")).toBeNull();
});

it("shows the alt text for a picture whose address the sanitiser removed, instead of a broken image", () => {
  const { container } = render(<Markdown text={'<img src="file:///C:/Users/me/a.jpg" alt="wall">'} local={refs("C:\\Work")} />);
  expect(container.querySelector("img")).toBeNull();
  expect(container.querySelector(".md-noimg")?.textContent).toBe("wall");
});
