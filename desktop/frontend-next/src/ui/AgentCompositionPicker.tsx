import { useContext, useEffect, useState } from "react";
import { CompositionNewContext } from "./CompositionNewContext";
import type { AgentPort } from "../port/port";
import type { AgentPresetCatalog, CompositionSummary } from "../port/composition";
import { t } from "../i18n";
import { reason } from "../i18n/kernel";

export function AgentCompositionPicker({ port, binding,locked:statusLocked,running, onChanged, onSettings, onError }: { port: AgentPort; binding?:CompositionSummary|null;locked?:boolean; running: boolean; onChanged: () => void; onSettings: () => void; onError: (e: unknown) => void }) {
  const [catalog, setCatalog] = useState<AgentPresetCatalog | null>(null);
  const [current, setCurrent] = useState<CompositionSummary | null>(null);
  const [busy, setBusy] = useState(false);
  const [next, setNext] = useState("");
  const createNew = useContext(CompositionNewContext);
  useEffect(() => {
    let active = true;
    Promise.all([port.agentPresets(), port.agentComposition()]).then(([list, binding]) => { if (active) { setCatalog(list); setCurrent(binding); } }).catch(() => {});
    return () => { active = false; };
  }, [port, running,binding?.digest,binding?.started]);
  if (!catalog) return null;
  const selected=catalog.presets.find((v)=>v.definition.id===current?.id)?.definition;
  const locked = running || statusLocked || !!current?.started;
  return <span className="composition-picker">
    <select className="mode" aria-label={t("Agent 组合")} data-action-change="agent-composition.select" disabled={busy || locked} value={current?.id ?? ""} title={locked ? t("该会话已开始，组合保持原版本。请用其他组合新建会话。") : selected?.description?t(selected.description):t("首次提交前可切换 Agent 组合")} onChange={(e) => {
      const id = e.target.value; setBusy(true);
      port.selectAgentComposition(id).then((binding) => { setCurrent(binding); onChanged(); }).catch((e) => onError(reason(e))).finally(() => setBusy(false));
    }}>
      {!current && <option value="">{t("原有会话配置")}</option>}
      {current && !catalog.presets.some((v) => v.definition.id === current.id) && <option value={current.id}>{current.name}</option>}
      {catalog.presets.map((v) => <option key={v.definition.id} value={v.definition.id}>{v.definition.id.startsWith("builtin:")?t(v.definition.name):v.definition.name} · {v.definition.id.split(":")[0]}</option>)}
    </select>
    {locked && createNew && <><select className="mode" aria-label={t("用其他组合新建会话")} data-action-change="agent-composition.new-choice" value={next} onChange={(e) => setNext(e.target.value)}><option value="">{t("用其他组合新建会话")}</option>{catalog.presets.map((v) => <option key={v.definition.id} value={v.definition.id}>{v.definition.id.startsWith("builtin:")?t(v.definition.name):v.definition.name}</option>)}</select><button className="mode" type="button" data-action="agent-composition.new" disabled={!next} onClick={() => createNew(next)}>{t("新建会话")}</button></>}
    {current?.unavailable && <span role="alert" title={current.unavailable}>{t("组合资源不可用，当前会话仅可查看历史。")}</span>}
    <button type="button" className="mode plain" data-action="agent-composition.manage" onClick={onSettings} aria-label={t("管理 Agent 预设")}>⋯</button>
  </span>;
}
