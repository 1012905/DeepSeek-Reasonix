import { useCallback, useEffect, useRef, useState } from "react";
import type { AgentPort, PeersState } from "../port/port";

const EMPTY: PeersState = { enabled: false, peers: [], held: [] };

export function usePeers(port: AgentPort, everyMs: number) {
  const [state, setState] = useState<PeersState>(EMPTY);
  const [failed, setFailed] = useState(false);
  const alive = useRef(true);
  const refresh = useCallback(async () => {
    try {
      const next = await port.peers();
      if (alive.current) {
        setState(next);
        setFailed(false);
      }
    } catch {
      if (alive.current) setFailed(true);
    }
  }, [port]);
  useEffect(() => {
    alive.current = true;
    void refresh();
    const timer = setInterval(() => {
      if (typeof document === "undefined" || !document.hidden) void refresh();
    }, everyMs);
    return () => {
      alive.current = false;
      clearInterval(timer);
    };
  }, [refresh, everyMs]);
  return { state, failed, refresh };
}
