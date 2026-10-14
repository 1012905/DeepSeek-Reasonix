// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { BackupFile } from "./BackupFile";
import { MockPort } from "../port/mock";
import type { AgentPort } from "../port/port";

afterEach(cleanup);

const passwordFields = () => document.querySelectorAll<HTMLInputElement>('input[type="password"]');

describe("backup to a file", () => {
  it("leaves API keys unticked, and sends the ticked categories with the passphrase to be sealed", async () => {
    const port = new MockPort() as unknown as AgentPort;
    const exporter = vi.spyOn(port, "exportBackupFile");
    const save = vi.spyOn(port, "saveText");
    render(<BackupFile port={port} />);

    const secrets = await screen.findByRole("checkbox", { name: /API 密钥/ });
    expect((secrets as HTMLInputElement).checked).toBe(false);
    const [pass, again] = passwordFields();
    await userEvent.type(pass!, "a long passphrase");
    await userEvent.type(again!, "a long passphrase");
    await userEvent.click(screen.getByRole("button", { name: "导出到文件…" }));

    await waitFor(() => expect(exporter).toHaveBeenCalledOnce());
    const [categories, passphrase] = exporter.mock.calls[0]!;
    expect(categories).not.toContain("secrets");
    expect(passphrase).toBe("a long passphrase");
    await waitFor(() => expect(save).toHaveBeenCalledWith("reasonix-config-demo.rxcb", "UlhDQgAAAA=="));
    expect(passwordFields()[0]!.value).toBe("");
  });

  it("holds the button back, and says why, until the passphrase is long enough and repeated", async () => {
    const port = new MockPort() as unknown as AgentPort;
    render(<BackupFile port={port} />);
    const button = await screen.findByRole<HTMLButtonElement>("button", { name: "导出到文件…" });
    expect(button.disabled).toBe(true);
    const [pass, again] = passwordFields();
    await userEvent.type(pass!, "abcd");
    expect(document.querySelector('[data-action="backup.wait"]')!.textContent).toBe("口令还差 6 个字符");
    await userEvent.type(pass!, "efghij");
    await userEvent.type(again!, "abcdefghiX");
    expect(button.disabled).toBe(true);
    expect(screen.getByText("两次输入的口令不一致")).toBeTruthy();
  });

  it("warns that ticked keys travel in the file", async () => {
    const port = new MockPort() as unknown as AgentPort;
    render(<BackupFile port={port} />);
    const secrets = await screen.findByRole("checkbox", { name: /API 密钥/ });
    await userEvent.click(secrets);
    expect(secrets.closest("label")!.hasAttribute("data-warn")).toBe(true);
  });

  it("opens a chosen file with its passphrase, previews it, and applies through the file route", async () => {
    const port = new MockPort() as unknown as AgentPort;
    const preview = vi.spyOn(port, "previewBackupFile");
    const apply = vi.spyOn(port, "applyBackupFile");
    render(<BackupFile port={port} />);
    await screen.findByRole("button", { name: "导出到文件…" });

    const input = document.querySelector<HTMLInputElement>('input[type="file"]')!;
    await userEvent.upload(input, new File(["  U0VBTEVE\n"], "laptop.rxcb"));
    await userEvent.type(await screen.findByLabelText("加密口令"), "a long passphrase");
    await userEvent.click(screen.getByRole("button", { name: "预览差异" }));

    await waitFor(() => expect(preview).toHaveBeenCalledWith("U0VBTEVE", "a long passphrase"));
    await userEvent.click(await screen.findByRole("button", { name: /恢复所选/ }));
    await waitFor(() => expect(apply).toHaveBeenCalled());
  });

  it("says nothing when the kernel does not offer files to this client", async () => {
    const port = new MockPort() as unknown as AgentPort;
    vi.spyOn(port, "backupFileCatalog").mockRejectedValue(new Error("a paired device cannot use this"));
    const view = render(<BackupFile port={port} />);
    await waitFor(() => expect(port.backupFileCatalog).toHaveBeenCalled());
    await waitFor(() => expect(view.container.textContent).toContain("本机窗口"));
    expect(screen.queryByRole("button", { name: "导出到文件…" })).toBeNull();
  });
});
