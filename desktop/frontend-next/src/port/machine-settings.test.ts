// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest";
import { SseHub } from "./hub";

afterEach(() => vi.unstubAllGlobals());

it("keeps machine reads and writes independent of runtime IDs and reconnects", async () => {
  const fetcher = vi.fn(async (_url: string, _init?: RequestInit) => new Response(JSON.stringify({ memories: [] })));
  vi.stubGlobal("fetch", fetcher);
  const hub = new SseHub();
  const machine = hub.machinePort();
  hub.portFor({ id: "r1", base: "/rt/r1", root: "/remote", name: "Remote", host: "gpu" });
  await machine.memories();
  await machine.forgetMemory("local-fact");
  hub.portFor({ id: "r2", base: "/rt/r2", root: "/local", name: "Local" });
  expect(hub.machinePort()).toBe(machine);
  await machine.memories();
  expect(fetcher.mock.calls.map(([url]) => url)).toEqual([
    "/host/settings/memory", "/host/settings/memory/forget", "/host/settings/memory",
  ]);
});
