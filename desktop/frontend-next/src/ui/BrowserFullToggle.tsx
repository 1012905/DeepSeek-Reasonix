import { t } from "../i18n";
import { StudioIcon } from "./StudioIcon";
import { ariaChord, chord } from "./keys";

export function BrowserFullToggle({ full, onToggle }: { full: boolean; onToggle: () => void }) {
  const label = full ? t("退出全屏") : t("全屏浏览器");
  return (
    <button className="workbench-files" data-action="browser.full" aria-pressed={full}
      aria-label={label} aria-keyshortcuts={`${ariaChord("F")}+Shift`}
      title={`${label} (${chord("Shift+F")})`} onClick={onToggle}>
      <StudioIcon name="expand" />
    </button>
  );
}
