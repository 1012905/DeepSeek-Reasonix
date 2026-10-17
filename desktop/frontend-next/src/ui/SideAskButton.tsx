import { forwardRef, useId } from "react";
import { t } from "../i18n";
import { StudioIcon } from "./StudioIcon";

/** The side question's switch, drawn beside attach. Whether the panel is open is
 *  the pane's to say; this only reports the press and the state it was given. */
export const SideAskButton = forwardRef<HTMLButtonElement, { open: boolean; onToggle: () => void }>(function SideAskButton({ open, onToggle }, ref) {
  const tip = useId();
  return (
    <button
      ref={ref}
      type="button"
      className="mode plain attach studio-attach"
      data-action="side.toggle"
      aria-label={t("辅助会话")}
      aria-describedby={tip}
      aria-pressed={open}
      onClick={onToggle}
    >
      <StudioIcon name="chat" />
      <span className="studio-control-tip" id={tip} role="tooltip">
        <b>{t("辅助会话")}</b>
        <span>{t("只读 · 不写入主会话 · 关闭即丢")}</span>
      </span>
    </button>
  );
});
