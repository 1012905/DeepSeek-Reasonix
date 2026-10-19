import type { AgentPresetCatalog, AgentPresetVersion } from "../port/composition";
import { t } from "../i18n";
import { StudioIcon } from "./StudioIcon";
import "../styles/composition-presets.css";

export function CompositionCatalog({ catalog, onSelect, onCreate, onCopy }: {
  catalog: AgentPresetCatalog;
  onSelect: (version: AgentPresetVersion) => void;
  onCreate: () => void;
  onCopy: (version: AgentPresetVersion) => void;
}) {
  return <div className="composition-catalog">
    <header className="composition-list-heading">
      <h2>{t("Agent 预设")}</h2>
      <button className="btn" data-primary="" data-action="agent-preset.create" onClick={onCreate}><StudioIcon name="plus" />{t("新建预设")}</button>
      <p>{t("为不同会话组合人设、工具、技能与扩展。模型保持独立；已开始的会话保留原版本。")}</p>
    </header>
    <p className="composition-default-note">{t("新会话默认使用标准模式。选择其他模式后才会启用对应预设。")}</p>
    {(["builtin", "user", "project"] as const).map((scope) => {
      const items = catalog.presets.filter(v => v.definition.id.startsWith(scope + ":"));
      if (!items.length) return null;
      return <section className="composition-list-group" key={scope} aria-label={t(scope === "builtin" ? "内置" : scope === "user" ? "个人" : "项目共享")}>
        <h3>{t(scope === "builtin" ? "内置" : scope === "user" ? "个人" : "项目共享")}</h3>
        {items.map(v => <PresetRow key={v.definition.id} version={v} onSelect={() => onSelect(v)} onCopy={() => onCopy(v)} />)}
      </section>;
    })}
    <details className="composition-how"><summary>{t("如何使用")}</summary><p>{t("在输入框顶部选择 Agent 模式，首次提交前可切换。开始对话后，请用其他模式新建会话。精简模式的终端仍可修改文件，并非只读模式。")}</p></details>
  </div>;
}

function PresetRow({ version: { definition: d }, onSelect, onCopy }: { version: AgentPresetVersion; onSelect: () => void; onCopy: () => void }) {
  const builtin = d.id.startsWith("builtin:");
  const name = builtin ? t(d.name) : d.name;
  return <article className="composition-row">
    <button className="composition-row-main" data-action="agent-preset.view" data-target={d.id}
      aria-label={t(builtin ? "查看 {name}" : "编辑 {name}", { name })} onClick={onSelect}>
      <span className="composition-row-icon"><StudioIcon name={d.id === "builtin:minimal" ? "code" : "layers"} /></span>
      <span className="composition-row-copy"><strong>{name}{d.id === "builtin:general" && <small>{t("默认")}</small>}</strong>
        <span>{builtin ? t(d.description) : d.description}</span></span>
      <StudioIcon name="chevron" />
    </button>
    <button className="act" data-action="agent-preset.copy" data-target={d.id} aria-label={t("复制 {name}", { name })} onClick={onCopy}>{t("复制")}</button>
  </article>;
}
