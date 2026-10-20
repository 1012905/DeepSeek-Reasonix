// @vitest-environment jsdom
import "./testkit";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it } from "vitest";
import { CompositionEditor } from "./CompositionEditor";
import { MockPort } from "../port/mock";
import type { AgentComposition, AgentPresetCatalog } from "../port/composition";

afterEach(cleanup);

async function open() {
  const port = new MockPort();
  const catalog: AgentPresetCatalog = await port.agentPresets();
  const draft: AgentComposition = structuredClone(catalog.presets.find((v) => v.definition.id.startsWith("user:") || true)!.definition);
  draft.id = "user:picker";
  draft.tools = { mode: "select", members: [] };
  let current = draft;
  const view = render(
    <CompositionEditor draft={current} modules={catalog.modules} disabled={false} creating
      onChange={(next: AgentComposition) => { current = next; view.rerender(<CompositionEditor draft={current} modules={catalog.modules} disabled={false} creating onChange={() => {}} onValidityChange={() => {}} />); }}
      onValidityChange={() => {}} />,
  );
  return { catalog, get saved() { return current; } };
}

it("shows what a module is for beside its name, and says nothing where its source says nothing", async () => {
  await open();
  const pick = screen.getAllByRole<HTMLSelectElement>("combobox").find((s) => /添加模块/.test(s.getAttribute("aria-label") ?? ""))!;
  const labels = [...pick.options].map((o) => o.textContent);
  expect(labels).toContain("bash — Run a shell command in the workspace and return its output.");
  expect(labels).toContain("task");
  const bash = [...pick.options].find((o) => o.value === "bash")!;
  expect(bash.title).toBe("Run a shell command in the workspace and return its output.");
});

it("stores only the id of the module it adds, never its description", async () => {
  const state = await open();
  const pick = screen.getAllByRole<HTMLSelectElement>("combobox").find((s) => /添加模块/.test(s.getAttribute("aria-label") ?? ""))!;
  await userEvent.selectOptions(pick, "bash");
  expect(state.saved.tools.members).toEqual([{ id: "bash" }]);
});
