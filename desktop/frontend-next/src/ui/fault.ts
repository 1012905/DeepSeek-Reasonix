// A TypeError, RangeError or ReferenceError reaching the error bar is a defect
// in this page, not something the person did or the kernel refused, and the bar
// shows only its message. This leaves the stack in the console so a report of
// such a bar can say where it came from. Only the error's name, its message cut
// short and the frames' own locations are written, never what a request carried.
const PAGE_FAULTS = [TypeError, RangeError, ReferenceError];
const MESSAGE_CAP = 200;
const FRAMES = 12;

export function noteFault(e: unknown): void {
  if (!(e instanceof Error) || !PAGE_FAULTS.some((kind) => e instanceof kind)) return;
  const frames = (e.stack ?? "").split("\n").filter((line) => /^\s+at /.test(line)).slice(0, FRAMES);
  console.error(`[reasonix] page fault ${e.name}: ${e.message.slice(0, MESSAGE_CAP)}\n${frames.join("\n")}`);
}
