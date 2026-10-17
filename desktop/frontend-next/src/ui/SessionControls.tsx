import type { AgentPort, SessionStatus } from "../port/port";
import { Policy } from "./Policy";
import { AgentCompositionPicker } from "./AgentCompositionPicker";

export function SessionControls({ port, status, running, onChanged, onSettings, onError }: { port: AgentPort; status: SessionStatus | null; running: boolean; onChanged: () => void; onSettings: (at: string) => void; onError: (e: unknown) => void }) {
  return <>
    <Policy port={port} status={status} onChanged={onChanged} onBoundary={() => onSettings("tools:sandbox")} />
    <AgentCompositionPicker port={port} binding={status?.agentComposition} locked={status?.agentCompositionLocked} running={running} onChanged={onChanged} onSettings={() => onSettings("persona")} onError={onError} />
  </>;
}
