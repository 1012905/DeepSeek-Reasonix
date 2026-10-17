import { useCallback, useEffect, useRef, useState } from "react";
import type { AgentPort, WorkspaceBranch, WorkspaceGit } from "../port/port";

export function useWorkspaceBranches(port: AgentPort, onChanged: () => void, onSwitched: ((git: WorkspaceGit) => void) | undefined, onError: (error: unknown) => void, root = "") {
  const [locals, setLocals] = useState<WorkspaceBranch[]>([]);
  const [moving, setMoving] = useState<string | null>(null);
  const [reading, setReading] = useState<"idle" | "loading" | "ready" | "failed">("idle");
  const generation = useRef(0);
  useEffect(() => { generation.current++; setLocals([]); setMoving(null); setReading("idle"); return () => { generation.current++; }; }, [port, root]);
  const loadBranches = useCallback(() => {
    const current = generation.current;
    setReading("loading");
    void port.branches().then((result) => { if (current === generation.current) { setLocals(result.branches); setReading("ready"); } }).catch((error) => { if (current === generation.current) { setReading("failed"); onError(error); } });
  }, [port, onError, root]);
  const switchTo = (name: string) => {
    if (moving !== null) return;
    const current = generation.current;
    setMoving(name);
    void port.switchBranch(name).then((git) => {
      if (current !== generation.current) return;
      onChanged(); onSwitched?.(git);
    }).catch((error) => { if (current === generation.current) onError(error); })
      .finally(() => { if (current === generation.current) setMoving(null); });
  };
  return { locals, moving, reading, loadBranches, switchTo };
}
