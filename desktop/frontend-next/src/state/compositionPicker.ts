import { useSyncExternalStore } from "react";

const KEY = "rx-show-agent-composition";
const listeners = new Set<() => void>();
let fallback = false;

function read(): boolean {
  try { return localStorage.getItem(KEY) === "on"; }
  catch { return fallback; }
}

function publish(): void {
  listeners.forEach((fn) => fn());
}

function refresh(event: StorageEvent): void {
  if (event.key === KEY || event.key === null) publish();
}

function subscribe(fn: () => void): () => void {
  listeners.add(fn);
  if (listeners.size === 1) window.addEventListener("storage", refresh);
  return () => {
    listeners.delete(fn);
    if (!listeners.size) window.removeEventListener("storage", refresh);
  };
}

export function setShowsCompositionPicker(on: boolean): void {
  fallback = on;
  try { localStorage.setItem(KEY, on ? "on" : "off"); }
  catch { /* retain the choice for this window */ }
  publish();
}

export function useShowsCompositionPicker(): boolean {
  return useSyncExternalStore(subscribe, read, read);
}
