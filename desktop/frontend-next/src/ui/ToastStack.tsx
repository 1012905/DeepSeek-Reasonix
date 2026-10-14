import { t } from "../i18n";
import { dismissToast, useToasts } from "../state/toasts";
import { heldDraft, holdDraft } from "./feedbackdraft";

// What failed is written into the feedback form's draft, appended to anything already there.
function report(text: string) {
  const draft = heldDraft();
  holdDraft({ ...draft, category: "bug", body: draft.body ? `${draft.body}\n\n${text}` : text });
}

/** The window's corner for failures that do not block what the person is doing.
 *  They stay until dismissed, and each can be reported from where it appeared. */
export function ToastStack({ onFeedback }: { onFeedback: () => void }) {
  const toasts = useToasts();
  if (toasts.length === 0) return null;
  return (
    <div className="toasts" role="region" aria-label={t("通知")}>
      {toasts.map((toast) => (
        <div className="toast" role="alert" key={toast.id}>
          <span>{toast.text}</span>
          <button data-action="toast.report" onClick={() => { report(toast.text); onFeedback(); dismissToast(toast.id); }}>{t("报告此问题")}</button>
          <button data-action="toast.dismiss" data-target={String(toast.id)} onClick={() => dismissToast(toast.id)}>{t("知道了")}</button>
        </div>
      ))}
    </div>
  );
}
