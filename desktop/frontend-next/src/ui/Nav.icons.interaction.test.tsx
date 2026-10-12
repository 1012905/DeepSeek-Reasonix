// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render } from "@testing-library/react";
import { Nav } from "./Nav";
import { ICON, type Section } from "./prefsnav";

const props = { at: null, onGo: vi.fn(), onHome: vi.fn(), onRail: vi.fn(), onFeedback: vi.fn(), feedbackUnread: 0, shown: true };

afterEach(cleanup);

const sectionMark = (section: Section) => render(<svg>{ICON[section]}</svg>).container.querySelector("svg")?.innerHTML;

describe("the icon rail draws the mark of the section it opens", () => {
  it.each([
    ["用量", "usage"],
    ["文件", "storage"],
    ["工具与权限", "tools"],
    ["扩展", "ext"],
    ["记忆", "memory"],
    ["远程", "remote"],
    ["账号", "account"],
    ["设置", "advanced"],
  ] as const)("%s shows the %s section mark", (label, section) => {
    const { container } = render(<Nav {...props} />);
    const btn = container.querySelector(`button[aria-label="${label}"]`);
    expect(btn).not.toBeNull();
    expect(btn?.querySelector("svg")?.innerHTML).toBe(sectionMark(section));
  });
});
