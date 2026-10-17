import { openLiveStream, type LiveStream } from "./livestream";
import type { MuxAttachRequest, MuxEnvelope, MuxOpened } from "./mux";

const UNKNOWN_CONNECTION = "mux.unknown_connection";
const RETRY_MS = 2_000;

/** What a pane gives the shared stream: where it stands, where its frames go,
 *  and how to open the stream of its own when the hub cannot carry it. */
export interface PaneFeed {
  cursor: () => number | null;
  onData: (raw: string) => void;
  own: () => LiveStream;
}

interface Pane {
  feed: PaneFeed;
  paused: boolean;
  own: LiveStream | null;
}

/** One EventSource for every local pane. A browser holds six connections to an
 *  origin; a stream per pane starved the window's own requests, so the hub puts
 *  every pane's frames on one stream, each in an envelope naming its pane. A
 *  pane keeps its own cursor and says it on every subscription, so a stream that
 *  dropped and came back under a new connection id resumes each pane where it
 *  stood. A pane the hub does not run locally keeps a stream of its own. */
export class MuxClient {
  private conn: string | null = null;
  private stream: LiveStream | null = null;
  private readonly panes = new Map<string, Pane>();
  private readonly listeners = new Set<() => void>();

  attach(id: string, feed: PaneFeed): LiveStream {
    const pane: Pane = { feed, paused: false, own: null };
    this.panes.set(id, pane);
    this.sync();
    void this.subscribe(id, pane);
    return Object.assign(
      () => {
        if (this.panes.get(id) === pane) this.panes.delete(id);
        if (pane.own) pane.own();
        else void this.unsubscribe(id);
        this.sync();
      },
      {
        pause: () => {
          if (pane.paused) return;
          pane.paused = true;
          if (pane.own) pane.own.pause();
          else void this.unsubscribe(id);
          this.sync();
        },
        resume: () => {
          if (!pane.paused) return;
          pane.paused = false;
          if (pane.own) pane.own.resume();
          else {
            this.sync();
            void this.subscribe(id, pane);
          }
        },
        holdsConnection: () => !!pane.own && !pane.paused,
      },
    );
  }

  /** Calls back when a pane has to leave the shared stream for one of its own. */
  onFallback(listener: () => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  /** Panes riding the shared stream, for tests and diagnostics. */
  carried(): string[] {
    return [...this.panes].filter(([, p]) => !p.own && !p.paused).map(([id]) => id);
  }

  private sync() {
    const needed = this.carried().length > 0;
    if (needed && !this.stream) {
      this.stream = openLiveStream(() => "/events/mux", (raw) => this.hear(raw));
    } else if (!needed && this.stream) {
      this.stream();
      this.stream = null;
      this.conn = null;
    }
  }

  private reopen() {
    this.stream?.();
    this.stream = null;
    this.conn = null;
    this.sync();
  }

  private hear(raw: string) {
    let msg: Partial<MuxOpened & MuxEnvelope>;
    try {
      msg = JSON.parse(raw);
    } catch {
      return;
    }
    if (msg.kind === "mux_open" && msg.conn) {
      this.conn = msg.conn;
      for (const [id, pane] of this.panes) void this.subscribe(id, pane);
      return;
    }
    if (!msg.pane) return;
    const pane = this.panes.get(msg.pane);
    if (!pane || pane.own || pane.paused) return;
    const frame = msg.frame as { kind?: string } | undefined;
    if (!frame || frame.kind === "mux_pane_closed") return;
    pane.feed.onData(JSON.stringify(frame));
  }

  private path(conn: string, id: string) {
    return `/events/mux/${conn}/panes/${encodeURIComponent(id)}`;
  }

  private async subscribe(id: string, pane: Pane) {
    const conn = this.conn;
    if (!conn || pane.own || pane.paused) return;
    const body: MuxAttachRequest = { after: pane.feed.cursor() };
    const live = () => this.conn === conn && this.panes.get(id) === pane && !pane.own && !pane.paused;
    try {
      const res = await fetch(this.path(conn, id), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify(body),
      });
      if (res.ok) return;
      const code = ((await res.json().catch(() => null)) as { code?: string } | null)?.code;
      if (res.status === 404 && code === UNKNOWN_CONNECTION) {
        if (this.conn === conn) this.reopen();
        return;
      }
      if (res.status === 409 || res.status === 404) {
        if (live()) this.fallBack(pane);
        return;
      }
    } catch {
      // The request never landed: asking again is the only repair.
    }
    setTimeout(() => {
      if (live()) void this.subscribe(id, pane);
    }, RETRY_MS);
  }

  private async unsubscribe(id: string) {
    const conn = this.conn;
    if (!conn) return;
    try {
      await fetch(this.path(conn, id), { method: "DELETE", credentials: "same-origin" });
    } catch {
      // The stream going away drops the subscription all the same.
    }
  }

  private fallBack(pane: Pane) {
    pane.own = pane.feed.own();
    this.sync();
    this.listeners.forEach((l) => l());
  }
}

export const mux = new MuxClient();
