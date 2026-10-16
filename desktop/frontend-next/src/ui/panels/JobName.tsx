import { useEffect, useId, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { createPortal } from "react-dom";
import type { JobEntry } from "../../port/port";

const LONG_PRESS_MS = 450;

/** What a job is called in the list: the name its caller gave it for a person,
 *  else the short preview of what it runs, else its id. */
export const jobName = (j: JobEntry) => j.description?.trim() || j.label || j.id;

/** The name of a job with its full text on hover, focus, Escape to close and a
 *  long press on a touch screen. The popover is real text inside the same wrapper,
 *  so a pointer can move onto it and select or copy the command. */
export function JobName({ job, children }: { job: JobEntry; children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const [place, setPlace] = useState<CSSProperties>({});
  const box = useRef<HTMLSpanElement>(null);
  const press = useRef<ReturnType<typeof setTimeout> | null>(null);
  const id = useId();
  const name = jobName(job);
  const command = job.detail?.trim() || (job.description?.trim() ? job.label : "");
  useEffect(() => () => { if (press.current) clearTimeout(press.current); }, []);
  // The list sits in a panel that clips what overflows it, and a later row would
  // paint over a tip drawn inside it, so the tip is placed in the window itself,
  // under the name or above it when there is no room below.
  useEffect(() => {
    if (!open || !box.current) return;
    const r = box.current.getBoundingClientRect();
    const width = Math.min(420, window.innerWidth * 0.8);
    const left = Math.max(8, Math.min(r.left, window.innerWidth - width - 8));
    setPlace(r.bottom + 120 > window.innerHeight ? { left, bottom: window.innerHeight - r.top + 4 } : { left, top: r.bottom + 4 });
  }, [open]);
  const stop = () => { if (press.current) { clearTimeout(press.current); press.current = null; } };
  return (
    <span
      className="job-name"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={(e) => { if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setOpen(false); }}
      onKeyDown={(e) => { if (e.key === "Escape" && open) { e.stopPropagation(); setOpen(false); } }}
      onPointerDown={(e) => { if (e.pointerType === "touch") { stop(); press.current = setTimeout(() => setOpen(true), LONG_PRESS_MS); } }}
      onPointerUp={stop}
      onPointerCancel={stop}
      aria-describedby={open ? id : undefined}
      ref={box}
    >
      {children}
      {open && createPortal(
        <span className="job-tip" role="tooltip" id={id} style={place}>
          <span className="jt-name">{name}</span>
          {command && command !== name && <code className="jt-cmd">{command}</code>}
        </span>,
        document.body,
      )}
    </span>
  );
}
