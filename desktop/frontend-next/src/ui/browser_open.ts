import { HttpError, type AgentPort, type BrowserTab } from "../port/port";

interface Opening { page: Promise<BrowserTab>; pending: boolean; started: boolean; order: number; tabId?: string; seen: Set<string> }
const opening = new WeakMap<AgentPort, Opening>();
const sequence = new WeakMap<AgentPort, number>();
const unseen = new WeakMap<AgentPort, Set<Opening>>();
const observed = new WeakMap<AgentPort, { ids: Set<string>; reconciled: boolean }>();

export function unseenPageOpenings(port: AgentPort, { startedOnly = false } = {}): Promise<BrowserTab>[] {
  return [...(unseen.get(port) ?? [])].filter((request) => !startedOnly || request.started).map((request) => request.page);
}

/** A refused navigation can still leave a tab. The wire names its public ID,
 *  not its browser target; unrelated errors carry no ownership here. */
export function failedPageID(error: unknown): string | undefined {
  if (!(error instanceof HttpError)) return;
  const { code, params } = error.reason ?? {};
  const id = params?.tab;
  return code && ["browser.open_failed", "browser.engine_missing", "browser.engine_failed", "browser.profile_busy"].includes(code)
    && typeof id === "string" && id.trim() ? id : undefined;
}

export function observePageTabs(port: AgentPort, tabs: BrowserTab[], reconciled = true): void {
  const ids = new Set(tabs.map((tab) => tab.id));
  observed.set(port, { ids, reconciled });
  for (const request of unseen.get(port) ?? []) {
    // A tab closing before the response cannot erase its earlier observation.
    for (const id of ids) request.seen.add(id);
    if (reconciled && request.tabId && request.seen.has(request.tabId)) unseen.get(port)?.delete(request);
  }
}

/** The one way this window asks the kernel for a tab. What is being opened is
 *  remembered until the kernel answers, so a column that needs a page to show
 *  takes that one rather than asking for another. */
export function openPage(port: AgentPort, url: string): Promise<BrowserTab> {
  return trackPage(port, port.browserOpen(url, true), true).page;
}

/** Reserve the whole user operation before workspace lookup or other awaits.
 *  Preparation may find an existing page instead of creating one. */
export function preparePage(port: AgentPort, prepare: () => Promise<string | BrowserTab>): Promise<BrowserTab> {
  const page = Promise.resolve().then(prepare).then((ready) => {
    if (typeof ready !== "string") return ready;
    markStarted(port, state);
    return port.browserOpen(ready, true);
  });
  const state = trackPage(port, page, false);
  return page;
}

function trackPage(port: AgentPort, page: Promise<BrowserTab>, started: boolean): Opening {
  const order = (sequence.get(port) ?? 0) + 1;
  sequence.set(port, order);
  const state: Opening = { page, pending: true, started, order, seen: new Set(observed.get(port)?.ids) };
  if (started) markStarted(port, state);
  const requests = unseen.get(port) ?? new Set<Opening>();
  unseen.set(port, requests);
  requests.add(state);
  const settled = (tabId?: string) => {
    state.pending = false;
    state.tabId = tabId;
    if (!tabId || (observed.get(port)?.reconciled && state.seen.has(tabId))) requests.delete(state);
  };
  page.then((tab) => settled(tab.id), (error) => settled(failedPageID(error)));
  return state;
}

function markStarted(port: AgentPort, state: Opening): void {
  state.started = true;
  // A slow preparation keeps its original intent order when it reaches the kernel.
  if ((opening.get(port)?.order ?? 0) < state.order) opening.set(port, state);
}

/** The page already on its way for this window, if one is: what a followed link
 *  asked for, which the column opened for it has to show. */
export function pageInFlight(port: AgentPort): Promise<BrowserTab> | null {
  const current = opening.get(port);
  return current?.pending && current.started ? newestPage(port, current.page) : null;
}

/** A page for a column that has none: the one already on its way, else a blank. */
export function pageForEmptyColumn(port: AgentPort): Promise<BrowserTab> {
  const current = opening.get(port);
  return newestPage(port, current?.pending && current.started ? current.page : openPage(port, "about:blank"));
}

async function newestPage(port: AgentPort, page: Promise<BrowserTab>): Promise<BrowserTab> {
  try {
    const tab = await page;
    const next = opening.get(port);
    return next?.started && next.page !== page ? newestPage(port, next.page) : tab;
  } catch (error) {
    const next = opening.get(port);
    if (next?.started && next.page !== page) return newestPage(port, next.page);
    throw error;
  }
}
