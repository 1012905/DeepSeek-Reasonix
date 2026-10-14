import { useSyncExternalStore } from "react";
import { t } from "../i18n";
import { onSessionQuickActionsChange, setShowsSessionQuickActions, showsSessionQuickActions } from "../state/sessionQuick";
import { Group } from "./Group";

export function SessionQuickSection() {
  const on = useSyncExternalStore(onSessionQuickActionsChange, showsSessionQuickActions, showsSessionQuickActions);
  return (
    <Group id="session-quick-actions" title={t("会话行快捷按钮")} hint={t("鼠标移到侧栏会话行上时，是否在右侧显示「归档」「删除」按钮。关闭时这两个操作仍在右键菜单里。默认关闭。")}>
      <div className="seg" data-text role="group" aria-label={t("会话行快捷按钮")}>
        <button data-action="appearance.session-quick-actions" data-value="off" aria-pressed={!on} onClick={() => setShowsSessionQuickActions(false)}>{t("关闭")}</button>
        <button data-action="appearance.session-quick-actions" data-value="on" aria-pressed={on} onClick={() => setShowsSessionQuickActions(true)}>{t("显示")}</button>
      </div>
    </Group>
  );
}
