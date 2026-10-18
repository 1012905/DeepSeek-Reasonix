import { useCallback, useEffect, useMemo, useState } from "react";
import { t } from "../i18n";
import { reason } from "../i18n/kernel";
import type { FeedbackBoard as Board, FeedbackBoardMe } from "../port/feedback";
import type { AgentPort } from "../port/port";

const newest = (a: string, b: string): number => {
  const pa = a.replace(/^v/, "").split(".").map(Number);
  const pb = b.replace(/^v/, "").split(".").map(Number);
  for (let i = 0; i < 3; i++) if ((pa[i] ?? 0) !== (pb[i] ?? 0)) return (pb[i] ?? 0) - (pa[i] ?? 0);
  return 0;
};

const failure = (e: unknown): string => reason(e);

// The public board of people whose suggestions shipped, and the person's own switch
// for being on it. Hidden where the service serves no board. The name is the only
// thing shown, and being listed is always the person's own, separate choice.
export function FeedbackBoard({ port }: { port: AgentPort }) {
  const [all, setAll] = useState<Board | null>(null);
  const [board, setBoard] = useState<Board | null>(null);
  const [release, setRelease] = useState("");
  const [signedIn, setSignedIn] = useState(false);
  const [me, setMe] = useState<FeedbackBoardMe | null>(null);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [problem, setProblem] = useState("");

  const load = useCallback(async () => {
    try {
      const first = await port.feedbackBoard();
      setAll(first);
      setBoard(first);
      const account = await port.account().catch(() => null);
      const on = !!account && account.signedIn && !account.expired;
      setSignedIn(on);
      if (on) setMe(await port.feedbackBoardMe().catch(() => null));
    } catch {
      setAll(null);
    }
  }, [port]);
  useEffect(() => { void load(); }, [load]);

  const versions = useMemo(
    () => [...new Set((all?.entries ?? []).flatMap((e) => e.releases.map((r) => r.version)))].sort(newest),
    [all],
  );

  const pick = async (next: string) => {
    setRelease(next);
    try {
      setBoard(await port.feedbackBoard(next || undefined));
    } catch (e) {
      setProblem(failure(e));
    }
  };

  const run = async (act: () => Promise<FeedbackBoardMe>) => {
    setBusy(true);
    setProblem("");
    try {
      setMe(await act());
      setAll(await port.feedbackBoard());
      setBoard(await port.feedbackBoard(release || undefined));
    } catch (e) {
      setProblem(failure(e));
    } finally {
      setBusy(false);
    }
  };

  if (!all?.available || !board) return null;
  const ranked = board.view === "release";
  const typed = draft.trim();

  return (
    <section className="fbk-board" aria-label={t("贡献榜")}>
      <div className="fbk-link-hd">
        <span className="fbk-link-name">{t("贡献榜")}</span>
        <span className="fbk-hint">{t("给反馈被采纳并发布的人。按版本看有名次，全部版本只按名字排序，不比高低。只有自己选择加入的人才会出现。")}</span>
      </div>
      <label className="fbk-board-pick">
        <span className="fbk-hint">{t("版本")}</span>
        <select aria-label={t("版本")} data-action-change="feedback.board-release" value={release} onChange={(e) => void pick(e.target.value)}>
          <option value="">{t("全部版本")}</option>
          {versions.map((v) => <option key={v} value={v}>{v}</option>)}
        </select>
      </label>
      {board.entries.length === 0 ? (
        <p className="fbk-hint">{t("这个版本还没有人上榜。")}</p>
      ) : (
        <ol className="fbk-board-list" data-ranked={ranked ? "" : undefined}>
          {board.entries.map((e) => (
            <li key={e.displayName}>
              {ranked && <span className="fbk-board-rank" aria-label={t("名次 {n}", { n: e.rank ?? 0 })}>{e.rank}</span>}
              <span className="fbk-board-name">{e.displayName}</span>
              <span className="fbk-board-credits">{t("{n} 条被采纳", { n: e.credits })}</span>
              {!ranked && (
                <span className="fbk-board-where">
                  {e.releases.slice(0, 3).map((r) => <span key={r.version} className="fbk-chip" data-status="recorded">{r.version}{r.count > 1 ? ` ×${r.count}` : ""}</span>)}
                  {e.releases.length > 3 && <span className="fbk-meta">+{e.releases.length - 3}</span>}
                </span>
              )}
            </li>
          ))}
        </ol>
      )}
      <p className="fbk-hint">{t("名次取决于维护者选择发布的内容：被婉拒的好反馈不计入。")}</p>

      <div className="fbk-board-me">
        <span className="fbk-link-name">{t("我在榜上")}</span>
        {!signedIn ? (
          <p className="fbk-hint">{t("先在设置里登录账号，才能决定是否上榜。")}</p>
        ) : me?.delisted ? (
          <p className="fbk-hint" data-tone="warn">{t("这个账号已被维护者从榜单移除，暂时不能再加入。")}</p>
        ) : me?.listed ? (
          <div className="fbk-link-row">
            <span className="fbk-chip" data-status="recorded">{t("榜上显示为")} · {me.displayName}</span>
            <button type="button" className="btn sm" data-action="feedback.board-leave" disabled={busy} onClick={() => void run(() => port.leaveFeedbackBoard())}>{t("退出榜单")}</button>
          </div>
        ) : (
          <>
            <p className="fbk-hint">{t("默认不上榜。加入后榜上只显示下面这个名字和采纳数，不显示邮箱、安装 ID 或路径；随时可以退出，退出后下一次刷新就不再出现。")}</p>
            <div className="fbk-link-row">
              <input className="fbk-board-name-field" data-action-change="feedback.board-name" aria-label={t("榜上显示的名字")} value={draft} maxLength={30} placeholder={t("榜上显示的名字")} onChange={(e) => setDraft(e.target.value)} />
              <button type="button" className="btn sm" data-primary data-action="feedback.board-join" disabled={busy || typed.length < 2} onClick={() => void run(() => port.joinFeedbackBoard(typed))}>{t("加入贡献榜")}</button>
            </div>
            {typed.length >= 2 && <p className="fbk-hint" aria-live="polite">{t("榜上将显示为：{name}", { name: typed })}</p>}
          </>
        )}
      </div>
      {problem && <p className="fbk-hint" role="alert" data-tone="error">{problem}</p>}
    </section>
  );
}
