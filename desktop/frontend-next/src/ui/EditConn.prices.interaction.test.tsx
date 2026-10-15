// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { EditConn } from "./EditConn";
import { MockPort } from "../port/mock";
import type { ProviderEdit, ProviderEntry } from "../port/port";

afterEach(() => { cleanup(); vi.restoreAllMocks(); });

const entry = (over: Partial<ProviderEntry> = {}): ProviderEntry => ({
  name: "gw", kind: "openai", baseUrl: "https://gw.example/v1", models: ["alpha", "beta"], default: "alpha", hasKey: true,
  keyEnv: "GW_KEY", inUse: false, preset: false, ...over,
} as ProviderEntry);

async function open(e: ProviderEntry) {
  const port = new MockPort();
  const edit = vi.spyOn(port, "editProvider").mockResolvedValue(undefined as never);
  render(<EditConn entry={e} port={port as never} busy="" setBusy={() => {}} onDone={() => {}} onRevert={() => {}} />);
  await screen.findByText("按模型设置价格");
  return { edit };
}
const field = (name: string) => screen.getByRole<HTMLInputElement>("textbox", { name });
const save = () => screen.getByRole<HTMLButtonElement>("button", { name: /保存/ });

describe("pricing a model on a connection", () => {
  it("saves a typed price for one model and leaves the other unpriced", async () => {
    const { edit } = await open(entry());
    fireEvent.change(field("alpha 的输入单价"), { target: { value: "3" } });
    fireEvent.change(field("alpha 的输出单价"), { target: { value: "15" } });
    fireEvent.change(field("alpha 的缓存命中单价"), { target: { value: "0.3" } });
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "alpha 的货币" }), "USD");
    await userEvent.click(save());
    await waitFor(() => expect(edit).toHaveBeenCalled());
    const sent = edit.mock.calls[0]![0] as ProviderEdit;
    expect(sent.modelPrices).toEqual({ alpha: { input: 3, output: 15, cacheHit: 0.3, currency: "USD" } });
  });

  it("blocks the save while a row has only one of input and output, and says why", async () => {
    await open(entry());
    fireEvent.change(field("beta 的输入单价"), { target: { value: "2" } });
    expect(screen.getByText("输入价和输出价需要都填写，才会保存这一行。")).toBeTruthy();
    expect(save().disabled).toBe(true);
    fireEvent.change(field("beta 的输出单价"), { target: { value: "8" } });
    expect(save().disabled).toBe(false);
  });

  it("shows a stored price and clears it when the row is emptied", async () => {
    const { edit } = await open(entry({ modelPrices: { alpha: { input: 3, output: 15, cacheHit: 0.3, currency: "USD" } } }));
    expect(field("alpha 的输入单价").value).toBe("3");
    expect(field("beta 的输入单价").value).toBe("");
    fireEvent.change(field("alpha 的输入单价"), { target: { value: "" } });
    fireEvent.change(field("alpha 的输出单价"), { target: { value: "" } });
    await userEvent.click(save());
    await waitFor(() => expect(edit).toHaveBeenCalled());
    expect((edit.mock.calls[0]![0] as ProviderEdit).modelPrices).toEqual({ alpha: null });
  });
});
