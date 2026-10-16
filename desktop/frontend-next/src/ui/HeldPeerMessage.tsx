import { useState } from "react";
import { t } from "../i18n";
import { reason } from "../i18n/kernel";
import type { AgentPort, PeerHeld } from "../port/port";
import { heldLabel } from "./peerKeys";

export function HeldPeerMessage({ port, held, onDone, compact }: { port: AgentPort; held: PeerHeld; onDone: () => void; compact?: boolean }) {
  const [busy, setBusy] = useState(false);
  const [open, setOpen] = useState(!compact);
  const [error, setError] = useState("");
  const decide = async (approve: boolean) => {
    setBusy(true);
    setError("");
    try {
      await port.resolvePeerMessage(held.id, approve);
      onDone();
    } catch (e) {
      setError(reason(e));
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="peerheld" role="group" aria-label={t("来自 {from} 的消息", { from: heldLabel(held) })}>
      <div className="peerrow">
        <span className="peerhead" title={t("来自另一个会话「{from}」的文字，不是你写的。批准后才会交给这个会话的模型。", { from: heldLabel(held) })}>
          {compact ? t("「{from}」发来一条消息，等你批准", { from: heldLabel(held) }) : t("来自另一个会话「{from}」的文字，不是你写的。批准后才会交给这个会话的模型。", { from: heldLabel(held) })}
        </span>
        <span className="acts">
          {compact && <button className="act" data-action="peers.expand" aria-expanded={open} onClick={() => setOpen(!open)}>{t(open ? "收起" : "展开")}</button>}
          <button className="act" data-action="peers.deny" disabled={busy} onClick={() => void decide(false)}>{t("拒绝")}</button>
          <button className="act" data-action="peers.approve" data-primary disabled={busy} onClick={() => void decide(true)}>{t("批准并交给模型")}</button>
        </span>
      </div>
      <pre className="peerbody" data-clamped={open ? undefined : ""}>{held.body}</pre>
      {error && <div className="why">{error}</div>}
    </div>
  );
}
