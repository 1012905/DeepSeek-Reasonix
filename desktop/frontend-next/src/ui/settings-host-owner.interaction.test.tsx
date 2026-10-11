// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { App } from "./App";
import { MockHub } from "../port/mock_hub";
import { MockPort } from "../port/mock";
import { rememberActivePane } from "./activepane";
import { boot, STORAGE } from "../i18n";
import type { MemoryCatalog } from "../port/port";

beforeEach(async () => { await import("./Settings"); localStorage.setItem(STORAGE, "zh"); boot(); });

afterEach(() => { cleanup(); vi.restoreAllMocks(); localStorage.clear(); sessionStorage.clear(); });

it("keeps machine reads separate from active-pane account operations", async () => {
  const hub = new MockHub(), local = new MockPort(), remote = new MockPort();
  hub.runtimes = async () => [{ id: "remote", base: "/rt/remote", host: "gpu", root: "/remote", name: "Remote" }];
  hub.portFor = () => remote;
  hub.machinePort = () => local;
  remote.providerSetup = async () => null;
  remote.welcomeSeen = async () => true;
  const facts: MemoryCatalog = { memories: [{ name: "local-fact", title: "This machine fact", activation: "relevant" }], recallQuery: "" };
  vi.spyOn(local, "memories").mockResolvedValue(facts);
  const remoteMemories = vi.spyOn(remote, "memories");
  const localAccount = vi.spyOn(local, "account");
  const remoteAccount = vi.spyOn(remote, "account").mockResolvedValue({ signedIn: true, user: { handle: "remote-user", label: "Remote account", email: "remote@example.test" } });
  const remoteLogout = vi.spyOn(remote, "accountLogout").mockResolvedValue();
  const localLogout = vi.spyOn(local, "accountLogout");
  render(<App hub={hub} />);
  await userEvent.click(await screen.findByRole("button", { name: /工具与集成/ }));
  await userEvent.click(await screen.findByRole("tab", { name: /记忆/ }, { timeout: 10000 }));
  await screen.findByText("This machine fact");
  expect(remoteMemories).not.toHaveBeenCalled();
  expect(localAccount).not.toHaveBeenCalled();
  expect(remoteAccount).toHaveBeenCalled();
  await userEvent.click(screen.getByRole("tab", { name: /账号/ }));
  await screen.findAllByText("Remote account");
  await userEvent.click(screen.getByRole("button", { name: "退出登录" }));
  await userEvent.click(screen.getByRole("button", { name: /确认退出/ }));
  expect(remoteLogout).toHaveBeenCalled();
  expect(localLogout).not.toHaveBeenCalled();
});


it.each(["second", "remote"])("uses the active local workspace, or first local for %s", async (active) => {
  const hub = new MockHub();
  const views = [
    { id: "first", base: "/rt/first", root: "/first", name: "First" },
    { id: "second", base: "/rt/second", root: "/second", name: "Second" },
    { id: "remote", base: "/rt/remote", root: "/remote", name: "Remote", host: "gpu" },
  ];
  const ports = new Map(views.map((rt) => [rt.id, new MockPort()]));
  const first = ports.get("first")!;
  const expected = active === "second" ? "second" : "first";
  for (const [id, port] of ports) {
    port.providerSetup = async () => null;
    port.welcomeSeen = async () => true;
    port.memories = async () => ({ memories: [{ name: id, title: `${id} fact`, activation: "relevant" }], recallQuery: "" });
    port.models = async () => ["first", "second"].map((ref) => ({ ref, model: ref, label: ref, provider: "fixture", default: ref === id }));
  }
  hub.runtimes = async () => views;
  hub.portFor = (rt) => ports.get(rt.id)!;
  hub.machinePort = (id?: string) => ports.get(id ?? "first")!;
  hub.pickFolder = async () => "/legacy-fixture";
  const imported = vi.spyOn(hub, "importLegacySessions");
  const forgot = vi.spyOn(ports.get(expected)!, "forgetMemory").mockResolvedValue();
  const storage = vi.spyOn(first, "storage");
  rememberActivePane(active);
  render(<App hub={hub} />);
  await userEvent.click(await screen.findByRole("button", { name: /工具与集成/ }));
  await userEvent.click(await screen.findByRole("tab", { name: /记忆/ }));
  await screen.findByText(`${expected} fact`);
  await userEvent.click(screen.getByRole("button", { name: "忘记" }));
  expect(forgot).toHaveBeenCalledWith(expected);
  await userEvent.click(screen.getByRole("tab", { name: /模型偏好/ }));
  await waitFor(() => expect((screen.getByRole("combobox", { name: "默认模型" }) as HTMLSelectElement).value).toBe(expected));
  await userEvent.click(screen.getByRole("tab", { name: /存储/ }));
  await userEvent.click(await screen.findByRole("button", { name: "选择旧版数据目录…" }));
  expect(imported).toHaveBeenCalledWith("/legacy-fixture", `/${expected}`);
  expect(storage).toHaveBeenCalled();
  if (active === "second") {
    await userEvent.keyboard("{Escape}");
    const tabs = within(screen.getByRole("tablist", { name: "会话面板" })).getAllByRole("tab");
    await userEvent.click(tabs[0]);
    await userEvent.click(screen.getByRole("button", { name: /工具与集成/ }));
    await userEvent.click(await screen.findByRole("tab", { name: /记忆/ }));
    await screen.findByText("first fact");
    expect(screen.queryByText("second fact")).toBeNull();
  }
});
