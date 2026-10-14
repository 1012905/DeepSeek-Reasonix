// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import "./testkit";
import { MeterRail } from "./MeterRail";

afterEach(cleanup);

function rail(firstToken: number) {
  return render(
    <MeterRail
      tps={0} trail={[]} running={false} speed={{ output: 80, modelSeconds: 2, average: 40, firstToken }}
      metrics={{ hit: 0, miss: 0 }} ctx={null} mcp={[]} compact={{ phase: "idle" } as never} cost="" wallet={{} as never}
      hideAmounts={false} tasks={[]} jobs={[]} onSettings={vi.fn()} onCancelJob={vi.fn(async () => {})}
    />,
  );
}

describe("the generation speed detail", () => {
  it("names the time to first token next to the other timings", () => {
    rail(1.25);
    const row = screen.getByText("首字时间").closest("div")!;
    expect(row.textContent).toContain("1.25s");
  });

  it("shows a dash rather than a zero when no round measured one", () => {
    rail(0);
    expect(screen.getByText("首字时间").closest("div")!.textContent).toContain("—");
  });
});
