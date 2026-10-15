import { useSyncExternalStore } from "react";
import type { UpdateMode, VersionHub } from "../port/version";

// In notify mode the window reads the version catalog in the background and
// remembers whether a release newer than this build is published. It installs
// nothing and shows nothing but a mark. The catalog read is the version panel's
// own call; a failure is silent and backs off, so an offline or proxied machine
// is not retried in a storm.
const FIRST_DELAY = 60_000;
const DAY = 24 * 60 * 60_000;
const RETRY_FIRST = 15 * 60_000;

export interface UpdateWatchPort {
  updateMode(): Promise<UpdateMode>;
  versions(): Promise<VersionHub>;
}

let available = "";
const listeners = new Set<() => void>();

const publish = (next: string) => {
  if (next === available) return;
  available = next;
  for (const listener of listeners) listener();
};

const subscribe = (listener: () => void) => {
  listeners.add(listener);
  return () => void listeners.delete(listener);
};

/** The newer release the last background read found, or "". */
export const readUpdateAvailable = (): string => available;

export function useUpdateAvailable(): string {
  return useSyncExternalStore(subscribe, readUpdateAvailable, readUpdateAvailable);
}

export function resetUpdateWatch(): void {
  publish("");
}

let active: UpdateWatch | null = null;

/** Tells the running check the mode changed, so the choice takes effect now. */
export function refreshUpdateWatch(): Promise<void> {
  return active ? active.refresh() : Promise.resolve();
}

export interface UpdateWatch {
  (): void;
  /** Reads the mode again and, in notify mode, checks now. */
  refresh(): Promise<void>;
}

export function startUpdateWatch(port: UpdateWatchPort): UpdateWatch {
  let timer: ReturnType<typeof setTimeout> | undefined;
  let stopped = false;
  let failures = 0;

  const later = (ms: number) => {
    if (stopped) return;
    clearTimeout(timer);
    timer = setTimeout(() => void check(), ms);
  };

  const check = async () => {
    let mode: UpdateMode;
    try {
      mode = await port.updateMode();
    } catch {
      failures++;
      later(Math.min(DAY, RETRY_FIRST * 2 ** (failures - 1)));
      return;
    }
    if (stopped) return;
    if (mode !== "notify") {
      publish("");
      failures = 0;
      later(DAY);
      return;
    }
    try {
      const hub = await port.versions();
      if (stopped) return;
      failures = 0;
      publish(hub.newer && !hub.pinned && !hub.withdrawn && hub.latest ? hub.latest : "");
      later(DAY);
    } catch {
      failures++;
      later(Math.min(DAY, RETRY_FIRST * 2 ** (failures - 1)));
    }
  };

  later(FIRST_DELAY);
  const stop = (() => {
    stopped = true;
    clearTimeout(timer);
    if (active === stop) active = null;
  }) as UpdateWatch;
  stop.refresh = async () => {
    failures = 0;
    clearTimeout(timer);
    await check();
  };
  active = stop;
  return stop;
}
