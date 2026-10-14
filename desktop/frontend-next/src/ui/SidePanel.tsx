import { useEffect, useRef, useState } from "react";
import { t } from "../i18n";
import { reason } from "../i18n/kernel";
import type { AgentPort } from "../port/port";
import { listenAction } from "./listen";

interface Turn {
  role: "me" | "side";
  text: string;
  quotes?: string[];
}

const QUOTE_CAP = 20000;

// A question asked beside the session: read-only, kept only for as long as this
// panel is open, and never part of the main conversation. The kernel opens the
// conversation at the first question and drops it when the panel closes.
export function SidePanel({ port, onClose }: { port: AgentPort; onClose: () => void }) {
  const [turns, setTurns] = useState<Turn[]>([]);
  const [quotes, setQuotes] = useState<string[]>([]);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const panel = useRef<HTMLElement>(null);
  const opened = useRef<Promise<string> | null>(null);
  const id = useRef("");

  useEffect(() => () => {
    const open = opened.current;
    if (open) void open.then((held) => port.closeSide(held)).catch(() => {});
  }, [port]);

  // Pressing the button collapses the selection in some browsers, so the last
  // passage selected outside this panel is kept as it is made.
  const lastSelected = useRef("");
  useEffect(() => {
    const remember = () => {
      const sel = window.getSelection();
      const text = sel?.toString().trim() ?? "";
      if (text && !(sel?.anchorNode && panel.current?.contains(sel.anchorNode))) lastSelected.current = text;
    };
    return listenAction(document, "selectionchange", { action: "side.quote", listener: remember });
  }, []);

  const quoteSelection = () => {
    const sel = window.getSelection();
    const live = sel?.toString().trim() ?? "";
    const text = live && !(sel?.anchorNode && panel.current?.contains(sel.anchorNode)) ? live : lastSelected.current;
    if (!text) {
      setError(t("先在对话里选中一段文字"));
      return;
    }
    lastSelected.current = "";
    setError("");
    setQuotes((all) => (all.includes(text) ? all : [...all, text.slice(0, QUOTE_CAP)]));
  };

  const send = async () => {
    const question = draft.trim();
    if (!question || busy) return;
    setBusy(true);
    setError("");
    try {
      opened.current ??= port.openSide().then((held) => { id.current = held; return held; });
      const held = await opened.current;
      const sent = quotes;
      const answer = await port.askSide(held, question, sent);
      setTurns((all) => [...all, { role: "me", text: question, quotes: sent }, { role: "side", text: answer }]);
      setDraft("");
      setQuotes([]);
    } catch (e) {
      opened.current = id.current ? opened.current : null;
      setError(reason(e));
    } finally {
      setBusy(false);
    }
  };

  const close = () => {
    const open = opened.current;
    opened.current = null;
    if (open) void open.then((held) => port.closeSide(held)).catch(() => {});
    onClose();
  };

  return (
    <aside className="sidepanel" ref={panel} role="complementary" aria-label={t("辅助会话")}>
      <header>
        <span><b>{t("辅助会话")}</b><small>{t("只读 · 不写入主会话 · 关闭即丢")}</small></span>
        <button type="button" data-action="side.close" aria-label={t("关闭辅助会话")} title={t("关闭辅助会话")} onClick={close}>×</button>
      </header>
      <div className="sidebody" aria-live="polite">
        {turns.length === 0 && <p className="sidehint">{t("在这里问关于这段代码、这个报错的问题：它只能读文件和搜索，不会改动任何东西。")}</p>}
        {turns.map((turn, i) => (
          <div key={i} className="sideturn" data-role={turn.role}>
            {turn.quotes?.map((q) => <blockquote key={q}>{q}</blockquote>)}
            <p>{turn.text}</p>
          </div>
        ))}
        {busy && <p className="sidehint">{t("正在读取并回答…")}</p>}
      </div>
      {error && <p className="sideerr" role="alert">{error}</p>}
      {quotes.length > 0 && (
        <ul className="sidequotes">
          {quotes.map((q, i) => (
            <li key={q}>
              <span>{q}</span>
              <button type="button" data-action="side.unquote" aria-label={t("移除引用 {n}", { n: i + 1 })} onClick={() => setQuotes((all) => all.filter((x) => x !== q))}>×</button>
            </li>
          ))}
        </ul>
      )}
      <footer>
        <button type="button" className="sidequote" data-action="side.quote" onClick={quoteSelection}>{t("引用选中内容")}</button>
        <textarea
          aria-label={t("向辅助会话提问")}
          value={draft}
          rows={2}
          readOnly={busy}
          data-action-change="side.draft"
          data-action-keydown="side.send"
          placeholder={t("提问…")}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
              e.preventDefault();
              void send();
            }
          }}
        />
        <button type="button" className="sidesend" data-action="side.send" disabled={busy || !draft.trim()} onClick={() => void send()}>{t("发送")}</button>
      </footer>
    </aside>
  );
}
