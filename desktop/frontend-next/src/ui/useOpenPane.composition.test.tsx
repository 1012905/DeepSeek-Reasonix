// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import type { AgentPort } from "../port/port";
import { HttpError } from "../port/http_error";
import type { HubPort, RuntimeView } from "../port/hub";
import { clearToasts, readToasts } from "../state/toasts";
import { useOpenPane } from "./useOpenPane";

afterEach(() => { cleanup(); clearToasts(); });

function blankPane() {
  const port = {
    agentPresets: vi.fn().mockRejectedValue(new Error("presets unavailable")),
    selectAgentComposition: vi.fn().mockResolvedValue(null),
  };
  const setActive = vi.fn();
  const hub = { open: vi.fn(), close: vi.fn() };
  const hook = renderHook(() => useOpenPane({
    hub: hub as unknown as HubPort,
    runtimes: [{ id: "blank", root: "/w" } as RuntimeView],
    panePorts: new Map([["blank", port as unknown as AgentPort]]),
    active: "other", reportsRef: { current: {} }, runsRef: { current: {} },
    setRuntimes: vi.fn(), setActive, setTakeover: vi.fn(), reloadPanes: vi.fn(),
    sw: { path: "", pending: false, opening: () => false, begin: () => 1,
      current: () => true, landed: vi.fn(), end: vi.fn(), cancel: vi.fn() },
  }));
  return { port, setActive, hub, open: async (agentComposition?: string) => {
    await act(async () => { await hook.result.current({ root: "/w", agentComposition }); });
  } };
}

it("keeps an existing blank pane's chosen mode without reading the catalog", async () => {
  const pane = blankPane();
  await pane.open("builtin:minimal");
  await pane.open();
  expect(pane.port.selectAgentComposition).toHaveBeenCalledExactlyOnceWith("builtin:minimal");
  expect(pane.port.agentPresets).not.toHaveBeenCalled();
  expect(pane.setActive).toHaveBeenLastCalledWith("blank");
  expect(pane.hub.open).not.toHaveBeenCalled();
});

it("still opens the pane and reports a refused explicit mode without rebuilding it", async () => {
  const pane = blankPane();
  pane.port.selectAgentComposition.mockRejectedValue(new HttpError(409, "locked", { code: "agent_composition.locked" }));
  await pane.open("builtin:minimal");
  expect(pane.setActive).toHaveBeenCalledWith("blank");
  expect(pane.hub.open).not.toHaveBeenCalled();
  expect(readToasts()).toHaveLength(1);
  expect(readToasts()[0].text).toContain("不能切换组合");
});
