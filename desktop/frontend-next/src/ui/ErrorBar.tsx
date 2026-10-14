import { t } from "../i18n";

/** A refused action's reason, kept in the window's corner until it is dismissed. */
export function ErrorBar({ text, onDismiss }: { text: string; onDismiss: () => void }) {
  return (
    <div className="errbar" role="alert">
      <span>{text}</span>
      <button onClick={onDismiss}>{t("知道了")}</button>
    </div>
  );
}
