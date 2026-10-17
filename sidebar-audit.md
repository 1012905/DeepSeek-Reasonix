---
status: active
owner: @esengine
backup: @SivanCola
reviewed: 2026-10-17
---

# Sidebar regression audit

Reviewed the current uncommitted sidebar changes above HEAD `3d983ca0f`.
This audit covers local and SSH project rows, hover actions, context menus,
inline rename, asynchronous mutations and keyboard navigation.

## Findings and repairs

| Priority | Reproduction | Repair |
| --- | --- | --- |
| P2 | Click local Archive twice while the first request waits: two callbacks run. | A shared per-session mutation owner rejects re-entry and disables the row actions until completion. |
| P2 | Start an SSH pin write, focus another control, then complete it: focus jumps back. | Restore focus only while the original action still owns focus or its removal left focus on the body. |
| P2 | Cancel an SSH deletion confirmation: focus is lost. | Capture the source row before replacement and find its current identity when restoring focus. |
| P2 | Press Shift+F10 or ContextMenu on a local conversation: no menu opens. | Route both keys through the same menu owner as right click, with row-relative placement. |
| P2 | Start inline rename: the input begins at x=65 under the leading pin. | Reserve the leading gutter and hide pin/archive actions during editing; text now starts at x=95. |
| P2 | Inspect archived or legacy SSH transcripts: neither attribute is identified. | Render the same archive marker and legacy explanation as local conversations. |
| P3 | Press Enter on a local row: both its handler and the parent tree invoke Open. | Stop propagation after the row handles activation. |

The first two and the rename overlap follow the new hover-action implementation.
The remaining findings are adjacent interaction gaps exposed by the unified list.

The duplicate Enter callback predates this audit and the application already
deduplicates its underlying open request. It was not a demonstrated duplicate
runtime or data-loss defect.

## Repair boundaries

`useSessionMutation` now owns pending operations and focus settlement for both
local and SSH sessions. Its in-flight guard is synchronous; display state is
only a projection. Archive removes the focused row without abandoning keyboard
navigation: focus moves to a surviving row, or the project toolbar when empty.

Local pin callbacks expose their completion promise so the same guard covers
their real asynchronous lifetime. Context menus and hover controls reach the
same operation owner. Opening an archiving row is blocked while its mutation
is pending; unrelated navigation remains available.

SSH flags remain kernel-owned and host-qualified. No protocol or persisted
format changed in this audit. The development fixture now also clears a pin
when archiving, matching `sessionstore.SetSessionArchived`.

## Evidence

Evidence set: `sidebar-audit/` in the private QA artifacts. Screenshots and
machine-readable results are retained locally and are not committed.

Deterministic delayed-promise tests failed before repair for duplicate local
archive, late SSH focus restoration, and cancelled SSH deletion. Separate
regressions failed for local keyboard menu keys, missing remote attributes,
and duplicate keyboard activation.

These represent ordinary network latency
and user input orderings, not injected private application states.

Browser evidence uses the existing development fixture:

1. `rename-before.png` and `rename-after.png`: same 316px sidebar crop, with
   the input uncovered after repair. `rename-geometry.json` records the input
   border, padding and hidden leading control.
2. `local-keyboard-menu.png`: Shift+F10 opens the session menu and focuses its
   first action; Escape returns to the original conversation.
3. `archive-focus.png`: archiving a focused local transcript moves focus to
   the next project row. Restoring the last archived transcript returns focus
   to the project toolbar. The test archive was restored.
4. `remote-delete-cancel.png`: cancelling SSH deletion returns focus to that
   conversation; no delete was submitted.
5. `remote-archived.png`: SSH archive metadata is visible and archiving clears
   the pin. The test archive was restored; no model request was sent.

The input-overlap images were inspected together. Normal, hover, editing and
menu states retain the existing fonts, neutral selection fill and icon assets.
This audit does not claim native Electron or a new real-SSH transport run.
Earlier real-SSH evidence is documented in [ssh-qa.md](ssh-qa.md).

## Verification

The final full frontend suite passed 5263 tests across 634 files. TypeScript,
the production build, repository lint and whitespace checks passed.

The Go transport code is unchanged by this
audit; no new live-SSH or native Electron qualification is claimed.

The final browser navigation reported no console errors or warnings. The
restored preview is running on its existing port, with the normal list active.

## Existing capability boundaries

- The selection command explicitly says local conversations; SSH bulk actions
  are not implemented. Local selection, select-all, cancellation and focus
  return were checked without applying a destructive batch operation.
- SSH Show more expands the loaded book. Its existing older-history message
  directs the reader to the far machine; complete remote pagination is not
  implemented by this audit.

## Retrospective

The reusable checks for leading action slots, asynchronous focus ownership and
row replacement were added to the local Reasonix development skill's desktop
UI guidance, outside this repository.

final result: passed
