import { useEffect, useState } from "react";
import { t } from "../i18n";
import { reason } from "../i18n/kernel";
import type { AgentPort, WriteLeaseSettings } from "../port/port";
import { Switch } from "./Switch";

// One switch over how sessions in one project wait on each other. Off is the
// strict default; on trades the whole-workspace hold for concurrency, says what
// that costs beside the switch, and says that only sessions opened afterwards
// pick it up: a running one keeps the lease it started with.
export function WriteLease({ port }: { port: AgentPort }) {
  const [state, setState] = useState<WriteLeaseSettings | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    port
      .writeLease()
      .then(setState)
      .catch(() => setState(null));
  }, [port]);

  if (!state) return <div className="empty">{t("无法读取多会话写入设置。")}</div>;

  const flip = async () => {
    setBusy(true);
    setError("");
    try {
      setState(await port.saveWriteLease(!state.optimistic));
    } catch (e) {
      setError(reason(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="box">
      <div className="lrow">
        <span className="tx">
          <span className="lb">{t("乐观模式（多会话并行）")}</span>
          <span className="ds">{t("没声明写入范围的操作（bash、MCP、没写 write_paths 的子代理）不再占住整个工作区，所以同一个项目里的多个会话可以同时工作，不必互相等待。代价：声明了路径的写入（如编辑文件）仍按路径互斥，但 bash 和 MCP 的写入没有事前的路径围栏，只在事后观察，两个会话用它们改同一个文件时可能互相覆盖。默认关闭（严格模式）；只对之后新开的会话生效。")}</span>
        </span>
        <Switch
          data-action="write-lease.optimistic"
          on={state.optimistic}
          busy={busy}
          label={t("乐观模式（多会话并行）")}
          onClick={() => void flip()}
        />
      </div>
      {error && <div className="why">{error}</div>}
    </div>
  );
}
