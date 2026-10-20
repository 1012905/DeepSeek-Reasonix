import { useCallback, useRef, type RefObject } from "react";
import { createPortal } from "react-dom";
import { useDismiss } from "../dismiss";
import { beside, pinToViewport, useFollow } from "../place";
import { StudioIcon, type StudioIconName } from "../StudioIcon";

export interface DeliveryAction {
  label: string;
  icon: StudioIconName;
  run: () => void;
}

export function DeliveryMenu({ anchor, label, actions, onClose }: {
  anchor: RefObject<HTMLButtonElement | null>; label: string;
  actions: DeliveryAction[]; onClose: () => void;
}) {
  const menu = useRef<HTMLDivElement>(null);
  const close = useCallback(() => {
    if (menu.current?.contains(document.activeElement)) anchor.current?.focus({ preventScroll: true });
    onClose();
  }, [anchor, onClose]);
  useDismiss(true, anchor, close, menu);
  const place = useCallback(() => {
    if (!anchor.current || !menu.current) return;
    const rect = anchor.current.getBoundingClientRect(), box = menu.current.getBoundingClientRect();
    pinToViewport(menu.current, rect.right - box.width, beside(rect, box.height, innerHeight, "below", 6));
  }, [anchor]);
  useFollow(true, place);
  return createPortal(<div className="delivery-menu" role="menu" aria-label={label} ref={menu}
    onKeyDown={(event) => {
      if (event.key === "Tab") { close(); return; }
      if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
      event.preventDefault();
      const items = [...event.currentTarget.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')];
      const at = items.indexOf(document.activeElement as HTMLButtonElement);
      const next = event.key === "Home" ? 0 : event.key === "End" ? items.length - 1
        : (at + (event.key === "ArrowDown" ? 1 : -1) + items.length) % items.length;
      items[next]?.focus();
    }}>
    {actions.map((action, index) => <button key={action.label} type="button" role="menuitem" autoFocus={index === 0}
      onClick={() => { close(); action.run(); }}>
      <StudioIcon name={action.icon} /><span>{action.label}</span>
    </button>)}
  </div>, document.body);
}
