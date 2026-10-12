import { useCallback, useEffect, useRef } from "react";
import type { AgentPort, BrowserTab } from "../port/port";
import { failedPageID, unseenPageOpenings } from "./browser_open";

/** A newer choice supersedes user-opened pages across both their response and
 *  tab-list notification. Unrelated agent pages remain eligible to follow. */
export function useBrowserSelection(port: AgentPort, select: (key: string) => void, tabs: BrowserTab[]) {
  const latestTabs = useRef(tabs);
  latestTabs.current = tabs;
  const revision = useRef(0);
  const ignored = useRef(new Set<Promise<BrowserTab>>());
  useEffect(() => {
    ignored.current = new Set();
    return () => { revision.current++; };
  }, [port]);
  const supersede = useCallback(() => {
    revision.current++;
    ignored.current = new Set(unseenPageOpenings(port));
  }, [port]);
  const current = useCallback(() => {
    const at = revision.current;
    return () => at === revision.current;
  }, []);
  const pick = useCallback((key: string) => {
    supersede();
    select(key);
  }, [supersede, select]);
  const follow = useCallback((key: string) => {
    const valid = current();
    // Snapshot before this tab-list batch retires its observed requests. A
    // later agent navigation of the same target is a different operation.
    const pending = unseenPageOpenings(port, { startedOnly: true }).filter((page) => ignored.current.has(page));
    if (!pending.length) { select(key); return; }
    const tabId = latestTabs.current.find((tab) => `browser:${tab.target}` === key)?.id;
    void Promise.allSettled(pending).then((results) => {
      const owned = results.some((result) => result.status === "fulfilled"
        ? `browser:${result.value.target}` === key
        : !!tabId && failedPageID(result.reason) === tabId);
      if (valid() && !owned) select(key);
    });
  }, [port, current, select]);
  return { pick, follow, supersede, current };
}
