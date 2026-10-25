// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { Appearance } from "./Appearance";
import { FLOW_KEY } from "./FlowResizer";
import { MockPort } from "../port/mock";
import { boot, STORAGE } from "../i18n";
import type { AgentPort } from "../port/port";

beforeEach(() => {
  localStorage.setItem(STORAGE, "zh");
  boot();
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
});

afterEach(() => {
  cleanup();
  localStorage.clear();
  vi.restoreAllMocks();
});

function panel(width: string) {
  return <Appearance port={new MockPort() as unknown as AgentPort} theme="light" onTheme={() => {}} contrast="" onContrast={() => {}}
    weight="" onWeight={() => {}} look={{ width } as never} onLook={() => {}} reloadThemes={() => {}} />;
}

it.each(["铺满窗口", "适宜阅读"])("choosing %s drops the width a hand dragged the reading column to", async (name) => {
  localStorage.setItem(FLOW_KEY, "900");
  render(panel(name === "铺满窗口" ? "standard" : "full"));
  await userEvent.click(await screen.findByRole("button", { name }));
  expect(localStorage.getItem(FLOW_KEY)).toBeNull();
});
