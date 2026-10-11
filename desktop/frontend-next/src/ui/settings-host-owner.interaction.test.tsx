// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { App } from "./App";
import { MockHub } from "../port/mock_hub";
import { MockPort } from "../port/mock";
import { boot, STORAGE } from "../i18n";
import type { MemoryCatalog } from "../port/port";

beforeEach(async () => { await import("./Settings"); localStorage.setItem(STORAGE, "zh"); boot(); });

afterEach(() => { cleanup(); vi.restoreAllMocks(); localStorage.clear(); });

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
