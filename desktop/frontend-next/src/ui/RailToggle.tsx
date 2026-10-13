import { t } from "../i18n";

export function RailToggle({ rail, onRail }: { rail: boolean; onRail: () => void }) {
  return (
    <button className="studio-menu" data-action="chrome.rail" onClick={onRail} aria-pressed={rail} aria-label={rail ? t("收起工作区栏") : t("展开工作区栏")}>
      <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M3 4h10M3 8h10M3 12h10" /></svg>
    </button>
  );
}
